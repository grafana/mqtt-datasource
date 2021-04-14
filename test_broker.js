const aedes = require('aedes')();
const server = require('net').createServer(aedes.handle);

const PORT = 1883;
const publishers = {};

const toMillis = {
  millisecond: ms => ms,
  second: sec => sec * 1000,
  minute: min => min * 60 * 1000,
  hour: hour => hour * 60 * 60 * 1000,
};

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
    console.log(`publishing to topic:`, topic);
    publishers[topic] = setInterval(() => {
      aedes.publish({
        topic,
        cmd: 'publish',
        qos,
        retain: false,
        payload: Math.random().toString(),
      });
    }, interval);
  }
};

server.listen(PORT, () => {
  console.log('server started and listening on port ', PORT);
  aedes.on('subscribe', subscriptions => subscriptions.forEach(createPublisher));
  aedes.on('connectionError', console.error);
  aedes.on('clientDisconnect', client => console.log(`disconnect: ${client.id}`));
  aedes.on('client', client => console.log(`connect: ${client.id}`));
});
