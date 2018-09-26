import { call, takeEvery, take, put, spawn, select } from 'redux-saga/effects'
import { eventChannel } from 'redux-saga'

function initChannel() {
    return eventChannel(emitter => {
        const ws = new WebSocket(`ws://${window.location.host}/feed`)

        ws.addEventListener('message', event => {
            try {
                const data = JSON.parse(event.data)
                return emitter({
                    type: 'FEED_DATA',
                    payload: {
                        data
                    }
                })
            } catch (e) {
                console.error('Could not parse message', event.data)
            }
        })

        return () => {
            ws.close()
        }
    })
}

function* feed() {
    const channel = yield call(initChannel)
    while(true) {
        const action = yield take(channel)
        yield put(action)
    }
}

function* processFeed({ payload }) {
    const { data } = payload

    switch(data.Operation) {
        case 'HeartBeat':
            const peer = yield select(state => state.peers[data.peerID])

            let tslBlock = Infinity
            let newTipset = data.Tags.heartbeat.HeaviestTipset.map(ts => ts['/']).sort()

            if (peer) {
                const ts1 = peer.tipset.join('')
                const ts2 = newTipset.join('')

                if (ts1 !== ts2) {
                    tslBlock = Date.now()
                } else {
                    tslBlock = peer.tslBlock
                }
            }

            yield put({
                type: 'SET_PEER_INFO',
                payload: {
                    peer: {
                        nick: data.peerName,
                        id: data.peerID,
                        tipset: newTipset,
                        height: data.Tags.heartbeat.TipsetHeight,
                        tslBlock: tslBlock,
                    }
                }
            })
        break;
    }
}

function* saga() {
    yield spawn(feed)
    yield takeEvery('FEED_DATA', processFeed)
}

export default saga;