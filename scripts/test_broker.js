const aedes = require('aedes')();
const fs = require('fs');
const path = require('path');

let server = require('net').createServer(aedes.handle);
let PORT = 1883;

if (process.argv[2] === 'tls') {
  const options = {
    key: fs.readFileSync(path.join(__dirname, '../testdata/server-key.pem')),
    cert: fs.readFileSync(path.join(__dirname, '../testdata/server-cert.pem')),
    ca: fs.readFileSync(path.join(__dirname, '../testdata/ca-cert.pem')),
    requestCert: true,
    rejectUnauthorized: true,
  };
  PORT = 8883;
  server = require('tls').createServer(options, aedes.handle);
}

const toMillis = {
  millisecond: (ms) => ms,
  second: (sec) => sec * 1000,
  minute: (min) => min * 60 * 1000,
  hour: (hour) => hour * 60 * 60 * 1000,
};

const publishers = {};
const createPublisher = ({ topic, qos }) => {
  let i = 0;
  const [duration, value] = topic.split('/');
  const fn = toMillis[duration];

  if (!fn || !value || value < 1) {
    console.log(`unknown interval:`, topic);
    return;
  }

  const interval = fn(value);

  if (!publishers[topic]) {
    publishers[topic] = setInterval(() => {
      let payload = Math.random();

      // use json object to test intervals less than 1 second
      if (interval % 1000 === 0) {
        payload = JSON.stringify({ a: payload, b: { c: { d: [payload] } } });
      }

      aedes.publish({
        topic,
        cmd: 'publish',
        qos,
        retain: false,
        payload: payload.toString(),
      });
    }, interval);
  }
};

server.listen(PORT, () => {
  console.log('server started and listening on port ', PORT);
  aedes.on('subscribe', (subscriptions) => subscriptions.forEach(createPublisher));
  aedes.on('connectionError', console.error);
  aedes.on('clientDisconnect', (client) => console.log(`disconnect: ${client.id}`));
  aedes.on('client', (client) => console.log(`connect: ${client.id}`));
});
