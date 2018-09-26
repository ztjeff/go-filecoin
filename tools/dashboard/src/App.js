import React, { Component } from 'react'
import { BrowserRouter as Router, Route, Switch } from 'react-router-dom'
import { connect } from 'react-redux';

import logo from './logo.svg'
import './App.css'

import Nav from './components/nav/nav'
import Network from './components/network/network'

const ConnectedNetwork = connect(
  (state) => {
    const peers = Object.values(state.peers)
    const lastBlockTime = peers.reduce((lbt, peer) => {
      if (peer.tslBlock < lbt) {
        return peer.tslBlock
      }
      
      return lbt
    }, Infinity)

    return {
      peers: peers,
      stats: {
        lastBlockTime,
        totalPeers: peers.length
      }
    }
  },
  (dispatch) => {
    return {}
  }
)(Network)


class App extends Component {
  render() {
    return (
      <Router>
        <div>
          <Nav />
          <div className="pv4 pr4" style={{paddingLeft: 60}}>
          <Switch>
            <Route exact path='/' component={ConnectedNetwork} />
          </Switch>
          </div>
        </div>
    </Router>
    );
  }
}

export default App;
