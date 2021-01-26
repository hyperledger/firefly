import { app } from '../common';
import request from 'supertest';
import assert from 'assert';

describe('Asset definitions - argument validation', async () => {

  it('Missing aggregation query', async () => {
    const result = await request(app)
      .post('/api/v1/assets/instances/aggregate')
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing or invalid aggregation query' });
  });

  it('Invalid aggregation query', async () => {
    const result = await request(app)
      .post('/api/v1/assets/instances/aggregate')
      .send({
        query: 'INVALID'
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing or invalid aggregation query' });
  });

  it('Aggregation unsupported with NeDB', async () => {
    const result = await request(app)
      .post('/api/v1/assets/instances/aggregate')
      .send({
        query: []
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Aggregation not supported in NeDB' });
  });

});