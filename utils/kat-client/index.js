'use strict';

const API_ENDPOINT = 'wss://zzkuwhhizm-zzigaja3ua.dev2-ws.photic.io/api/v1';
const KAT_DESTINATION = 'kld://app2app/z/dev2/m/zzxknul1nt/e/zzkuwhhizm/s/zzigaja3ua/d/a';
const CLIENT_DESTINATION = 'kld://app2app/z/dev2/m/zzxknul1nt/e/zzkuwhhizm/s/zzigaja3ua/d/external';
const APP_CREDENTIAL_USER = 'zzbvccppd3';
const APP_CREDENTIAL_PASSWORD = '_WOhsqQ6ee8pW39Surq9XkZvP4Al-hjDNoFaKUQQF20';

const socket = require('socket.io-client').connect(API_ENDPOINT,
  {
    extraHeaders: {
      Authorization: 'Basic ' + Buffer.from(APP_CREDENTIAL_USER + ':' + APP_CREDENTIAL_PASSWORD).toString('base64')
    }
  })
  .on('connect', () => {
    console.log('Consumer connected.')
    socket.emit('subscribe', [CLIENT_DESTINATION], (err, result) => {
      if (err) {
        console.log(err);
      } else {
        console.log(result);
      }
    });
  }).on('disconnect', () => {
    console.log('Consumer disconnected.');
  }).on('exception', exception => {
    console.log('Exception: ' + exception);
  }).on('error', err => {
    console.log('Error: ' + err);
  }).on('connect_error', err => {
    console.log('Connection error: ' + err);
  }).on('data', (message, key, timestamp) => {

    const data = JSON.parse(message.content);

    socket.emit('produce', {
      headers: {
        to: KAT_DESTINATION,
        from: CLIENT_DESTINATION
      },
      content: JSON.stringify({
        authorizationID: data.authorizationID,
        authorized: true
      })
    })

    // console.log('Message from: ' + message.headers.from);
    // console.log('Content: ' + message.content);
    // console.log('key: ' + key);
    // console.log('timestamp: ' + timestamp);
  });
