import React from 'react'

export default class ElapseTime extends React.Component {
    constructor() {
        super()

        this.timer = setInterval(() => this.forceUpdate(), 250)
    }
    componentWillUnmount() {
        clearInterval(this.timer)
    }
    render() {
        return (
            <span>{((Date.now() - this.props.start) / 1000).toFixed(1)}</span>
        )
    }
}
