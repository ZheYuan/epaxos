// TODO persistent log
package replica

import (
	"os"

	golog "github.com/coreos/go-log/log"
)

// instId is an id of the already updated instance
// messageChan is a toy channel for emulating broadcast
// TODO: return error

var log = golog.NewSimple(
	golog.PriorityFilter(
		golog.PriErr,
		golog.WriterSink(os.Stdout, golog.BasicFormat, golog.BasicFields),
	),
)

func (r *Replica) sendAccept(repId int, insId InstanceIdType, messageChan chan Message) {
	inst := r.InstanceMatrix[repId][insId]
	if inst == nil {
		// shouldn't get here
		log.Error("shouldn't get here, repId = ", repId, " insId = ", insId)
	}
	inst.status = accepted

	// TODO: persistent store the status
	accept := &Accept{
		cmds: inst.cmds,
		//seq:   inst.seq,
		deps:   inst.deps,
		repId:  repId,
		insId:  insId,
		ballot: inst.ballot,
	}

	// TODO: handle timeout
	for i := 0; i < r.N/2; i++ {
		go func() {
			messageChan <- accept
		}()
	}
}

func (r *Replica) recvAccept(ac *Accept, messageChan chan Message) {
	inst := r.InstanceMatrix[ac.repId][ac.insId]

	if inst == nil {
		r.InstanceMatrix[ac.repId][ac.insId] = &Instance{
			cmds: ac.cmds,
			//seq: inst.seq,
			deps:   ac.deps,
			ballot: ac.ballot,
			status: accepted,
			info:   new(InstanceInfo),
		}
		inst = r.InstanceMatrix[ac.repId][ac.insId] // for reference in below
	} else {
		if inst.status >= accepted || ac.ballot < inst.ballot {
			// return nack with status
			ar := &AcceptReply{
				ok:     false,
				repId:  ac.repId,
				insId:  ac.insId,
				ballot: inst.ballot,
				status: inst.status,
			}
			log.Debug("recvAccept: return nack")
			r.sendAcceptReply(ar, messageChan)
			return
		} else {
			inst.cmds = ac.cmds
			//inst.seq = ac.seq
			inst.deps = ac.deps
			inst.status = accepted
			inst.ballot = ac.ballot
		}
	}

	// reply OK
	ar := &AcceptReply{
		ok:     true,
		ballot: inst.ballot,
		repId:  ac.repId,
		insId:  ac.insId,
	}
	log.Debug("recvAccept: return ok")
	r.sendAcceptReply(ar, messageChan)
}

func (r *Replica) sendAcceptReply(ar *AcceptReply, messageChan chan Message) {
	messageChan <- ar
}

func (r *Replica) recvAcceptReply(ar *AcceptReply, messageChan chan Message) {
	inst := r.InstanceMatrix[ar.repId][ar.insId]
	if inst == nil {
		// TODO: should not get here
		log.Error("shouldn't get here, repId = ", ar.repId, " insId = ", ar.insId)
	}

	if inst.status > accepted {
		// we've already moved on, this reply is a delayed one
		// so just ignore it
		log.Warning("recvAcceptReply: receive an AcceptReply from an out-dated replica, means there must be a partition or recover")
		return
	}

	if !ar.ok {
		// there must be another proposer, so let's keep quiet
		log.Debug("recvAcceptReply: receive an AcceptReply with ok = false")
		return
	}

	if ar.ok {
		inst.info.acceptOkCnt++
		if inst.info.acceptOkCnt >= (r.N / 2) {
			// ok, let's try to send commit
			log.Debug("recvAcceptReply: enough replies, now try commit")
			r.sendCommit(ar.repId, ar.insId, messageChan)
		}
	}
}

func (r *Replica) sendCommit(repId int, insId InstanceIdType, messageChan chan Message) {
	inst := r.InstanceMatrix[repId][insId]
	if inst == nil {
		// shouldn't get here
		log.Error("shouldn't get here, repId = ", repId, " insId = ", insId)
	}

	inst.status = committed
	// TODO: persistent store

	// make commit message and send to all
	cm := &Commit{
		cmds: inst.cmds,
		//seq: inst.seq,
		deps:   inst.deps,
		repId:  repId,
		insId:  insId,
		ballot: inst.ballot,
	}
	for i := 0; i < r.N-1; i++ {
		go func() {
			messageChan <- cm
		}()
	}
}

func (r *Replica) recvCommit(cm *Commit) {
	inst := r.InstanceMatrix[cm.repId][cm.insId]
	if inst == nil {
		r.InstanceMatrix[cm.repId][cm.insId] = &Instance{
			cmds: cm.cmds,
			//seq: cm.seq,
			deps:   cm.deps,
			status: committed,
			ballot: cm.ballot,
			info:   new(InstanceInfo),
		}
	} else {
		if inst.status >= committed || cm.ballot < inst.ballot {
			// ignore the message
			log.Debug("recvCommit: ignore the Commit message")
			return
		}
	}
}
