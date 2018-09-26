const WebSocket = require('ws')
const net = require('net')
const readline = require('readline')
const PassThrough = require('stream').PassThrough

const agro = new PassThrough()
const linereader = readline.createInterface({
    input: agro
})

const wss = new WebSocket.Server({
    port: 5050
})

let nodes = []
let clients = []

const server = net.createServer(conn => {
    console.log("New log connection")
    nodes.push(conn)

    conn.on('data', data => {
        agro.write(data)
    })

    conn.on('close', () => {
        console.log("Closed log connection")
        nodes = nodes.filter(c => c != conn)
    })
}).listen(5000)

wss.on('connection', conn => {
    console.log("New dashboard connection")
    clients.push(conn)

    conn.on('close', () => {
        console.log("Closed dashboard connection")
        clients = clients.filter(c => c != conn)
    })
})

linereader.on('line', line => {
    clients.forEach(c => c.send(line))
})