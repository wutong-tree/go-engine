package anet

import (
	"go-engine/alog"
	"log"
	"math"
	"sync"
	"time"
)

type DefNetIOCallback = func(msg *PackHead)

var netIOCallbackMap sync.Map

func registCallback(head *PackHead, cb DefNetIOCallback) {
	//netIOCallbackMap.Store(head.SequenceID,cb)
	registCallbackWithinTimeLimit(head, cb, 0, nil)
}
func registCallbackWithinTimeLimit(head *PackHead, cb DefNetIOCallback, delayMillisecond int64, evtChan chan *PackHead) {
	createTime := time.Now().UnixNano() / int64(time.Millisecond)
	regist := &netIORegisterCallback{cb: cb, createTime: createTime, deadline: createTime + delayMillisecond, eventChan: evtChan}
	netIOCallbackMap.Store(head.SequenceID, regist)
}
type netIORegisterCallback struct {
	cb DefNetIOCallback
	//millisecond
	createTime int64
	//deadline, unit is millisecond.set a huge number if endless callback is needed.
	deadline int64
	//if a request has timeout callback, write to this chan to trigger the callback event.
	eventChan chan *PackHead
}


/*
 * return nil if not found.
 * todo, we should record succeed and failed request, at least the request count.
 */
func popCallback(head *PackHead) DefNetIOCallback {
	if v, ok := netIOCallbackMap.Load(head.SequenceID); ok {
		//var cb DefNetIOCallback
		if regist, ok := v.(*netIORegisterCallback); ok {
			netIOCallbackMap.Delete(head.SequenceID)
			if regist.cb != nil {
				return regist.cb
			}
			if regist.eventChan != nil {
				currentTime := time.Now().UnixNano() / int64(time.Millisecond)
				if regist.deadline >= currentTime {	//normal task finished on time.
					tmp := make([]byte, len(head.Body))
					copy(tmp, head.Body)
					head.Body = tmp
					regist.eventChan <- head
				} else {
					//overtime task, log it and abandon it.
					alog.Infof("overtime task, currentTime:%s, deadline:%d, createTime:%d", currentTime, regist.deadline,regist.createTime)
				}
			}
		} else {
			log.Println("type convert error for net callback")
		}

	}

	//netIOCallbackMap.Range(func(key, value interface{}) bool {
	//	fmt.Println(key,value)
	//	return true
	//})

	return nil
}

const startIndexForSequenceId = 1000000

var autoIncreaseSequenceId uint32 = startIndexForSequenceId
var autoIncreaseSequenceIdLocker = new(sync.Mutex)

func allocateNewSequenceId() uint32 {
	autoIncreaseSequenceIdLocker.Lock()
	if autoIncreaseSequenceId > math.MaxUint32-1 {
		autoIncreaseSequenceId = startIndexForSequenceId
	}
	autoIncreaseSequenceId++
	defer autoIncreaseSequenceIdLocker.Unlock()
	return autoIncreaseSequenceId
}
