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
  if (topic == "imu") {
    let i = 0;
    console.log(`publishing to topic:`, topic);
    publishers[topic] = setInterval(() => {
      var j = {
        "sample": i, 
        "properties": ["acceleration", "velocity"], 
        "sensors": ["accelerometer", "gyroscope"], 
        "axes": ["x", "y", "z"], 
        "sensor": {
          "accelerometer": {
            "x": Math.random(), 
            "y": Math.random(), 
            "z": Math.random()
          }, 
          "gyroscope": {
            "x": Math.random(), 
            "y": Math.random(), 
            "z": Math.random()
          }
        }, 
        "axis": {
          "x": {
            "accelerometer": Math.random(), 
            "gyroscope": Math.random()
          }, 
          "y": {
            "accelerometer": Math.random(), 
            "gyroscope": Math.random()
          }, 
          "z": {
            "accelerometer": Math.random(), 
            "gyroscope": Math.random()
          }
        }
      };
      var out = JSON.stringify(j);
      aedes.publish({
        topic,
        cmd: 'publish',
        qos,
        retain: false,
        payload: out,
      });
      i+=1;
    }, 1000);
    return
  }

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
