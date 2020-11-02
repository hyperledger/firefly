import { app } from '../common';
import request from 'supertest';
import assert from 'assert';

describe('Argument validation', async () => {

  it('Attempting to add an asset definition without a name should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/assets/definitions')
      .send({
        author: '0x0000000000000000000000000000000000000001',
        isContentPrivate: false
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing or invalid asset definition name' });
  });

  it('Attempting to add an asset definition without an author should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/assets/definitions')
      .send({
        name: 'Undescribed - unstructured',
        isContentPrivate: false
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing asset definition author' });
  });

  it('Attempting to add an asset definition without indicating if the content should be private or not should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/assets/definitions')
      .send({
        name: 'Undescribed - unstructured',
        author: '0x0000000000000000000000000000000000000001'
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing asset definition content privacy' });
  });

});
