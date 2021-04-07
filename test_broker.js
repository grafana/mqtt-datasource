const aedes = require('aedes')();
const server = require('net').createServer(aedes.handle);

const PORT = 1883;
const ONE_SECOND = 1_000;

server.listen(PORT, () => {
  console.log('server started and listening on port ', PORT);
  setInterval(() => {
    aedes.publish({ topic: 'test', message: Math.random() });
  }, ONE_SECOND);
});
