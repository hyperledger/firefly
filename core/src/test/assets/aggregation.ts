import { app } from '../common';
import request from 'supertest';
import assert from 'assert';

describe('Aggregation', async () => {

  it('Missing aggregation query', async () => {
    const result = await request(app)
      .post('/api/v1/assets/aggregate/1111')
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing or invalid aggregation query' });
  });

  it('Invalid aggregation query', async () => {
    const result = await request(app)
      .post('/api/v1/assets/aggregate/1111')
      .send({
        query: 'INVALID'
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing or invalid aggregation query' });
  });

  it('Aggregation not supported in NeDB', async () => {
    const result = await request(app)
      .post('/api/v1/assets/aggregate/1111')
      .send({
        query: []
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Aggregation not supported in NeDB' });
  });

});