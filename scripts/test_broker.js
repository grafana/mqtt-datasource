import { Aedes } from 'aedes';
import { readFileSync } from 'node:fs';
import { join } from 'path';
import { createServer } from 'node:net';
import { createServer as createServerTLS  } from 'node:tls';

const broker = await Aedes.createBroker()
let server = createServer(broker.handle);
let PORT = 1883;

if (process.argv[2] === '--tls') {
  const options = {
    key: readFileSync(join(import.meta.dirname, '../testdata/server-key.pem')),
    cert: readFileSync(join(import.meta.dirname, '../testdata/server-cert.pem')),
    ca: readFileSync(join(import.meta.dirname, '../testdata/ca-cert.pem')),
    requestCert: true,
    rejectUnauthorized: true,
  };
  PORT = 8883;
  server = createServerTLS(options, broker.handle);
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

  const parts = topic.split('/');
  const [duration, value] = [parts[parts.length - 2], parts[parts.length - 1]];
  const fn = toMillis[duration];

  if (!fn || !value || value < 1) {
    console.log(`unknown interval:`, topic);
    return;
  }

  const interval = fn(value);

  if (!publishers[topic]) {
    console.log('creating publisher for', topic, 'with interval', interval, 'ms');
    publishers[topic] = setInterval(() => {
      let payload = Math.random();

      // use json object to test intervals less than 1 second
      if (interval % 1000 === 0) {
        payload = JSON.stringify({ a: payload, b: { c: { d: [payload] } } });
      }

      broker.publish({
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
  broker.on('subscribe', (subscriptions) => subscriptions.forEach(createPublisher));
  broker.on('connectionError', console.error);
  broker.on('clientDisconnect', (client) => console.log(`disconnect: ${client.id}`));
  broker.on('client', (client) => console.log(`connect: ${client.id}`));
});
