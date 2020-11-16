const express = require('express');
const bodyParser = require('body-parser');
const app = express();
const port = 4000;

app.use(bodyParser.urlencoded({ extended: true }));
app.use(bodyParser.json());

app.get('/', (req, res) => {
  console.log(`Authorization request received:\n\n${JSON.stringify(req.body, undefined, 2)}`);
  res.send({ authorized: true });
})

app.listen(port, () => {
  console.log(`Authorization webhook running on port ${port}`);
});
