import React from 'react'
import ElapseTime from '../ElapseTime'

function Stats({totalPeers, lastBlockTime}) {
  return (
    <div className="monospace">
      <div className="cf">
        <dl className="fl fn-l w-50 dib-l w-auto-l lh-title mr5-l">
          <dd className="f6 fw4 ml0">Total Peers</dd>
          <dd className="f3 fw6 ml0">{totalPeers}</dd>
        </dl>
        <dl className="fl fn-l w-50 dib-l w-auto-l lh-title">
          <dd className="f6 fw4 ml0">Last Block</dd>
          <dd className="f3 fw6 ml0"><ElapseTime start={lastBlockTime}/>s ago</dd>
        </dl>
      </div>
    </div>
  );
}

export default Stats
