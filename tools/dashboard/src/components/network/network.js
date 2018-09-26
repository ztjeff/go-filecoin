import React from 'react'
import { Table, Th, Td } from '../table/network'
import Stats from '../stats/stats'
import ElapseTime from '../ElapseTime'

const Tipset = ({set}) => {
    return (
        <ul className="list ma0 pa0">
            {set.map(tip => <li key={tip} className="truncate">{tip}</li>)}
        </ul>
    )
}

const PeerRow = ({peer}) => {
    return (
        <tr className="striped--near-white2">
            <Td style={{ display: 'flex', alignItems: 'center' }}>{peer.nick}(<span className="truncate">{peer.id}</span>)</Td>
            <Td>
                <Tipset set={peer.tipset} />
            </Td>
            <Td>{peer.height}</Td>
            <Td><ElapseTime start={peer.tslBlock} />s</Td>
        </tr>
    )
}

const Network = ({ stats, peers }) => {
    return (
        <div>
            <section className="pv3">
                <Stats {...stats} />
            </section>
            <section className="pv3">
                <Table>
                    <thead>
                        <tr>
                            <Th>Peer</Th>
                            <Th>Tipset</Th>
                            <Th style={{ width: 100 }}>Height</Th>
                            <Th style={{ width: 110 }}>Last Block</Th>
                        </tr>
                    </thead>
                    <tbody>
                        {peers.map(peer => <PeerRow key={peer.id} peer={peer}/>)}
                    </tbody>
                </Table>
            </section>
        </div>
    )
}

export default Network