import { app } from '../../common';
import request from 'supertest';
import assert from 'assert';

export const testAssetArgumentValidation = () => {
describe('Asset definitions - argument validation', async () => {
  it('Attempting to get an asset definition that does not exist should raise an error', async () => {
    const result = await request(app)
      .get('/api/v1/assets/definitions/1000000')
      .expect(404);
    assert.deepStrictEqual(result.body, { error: 'Asset definition not found' });
  });

  it('Attempting to add an asset definition without a name should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/assets/definitions')
      .send({
        author: '0x0000000000000000000000000000000000000001',
        isContentPrivate: false,
        isContentUnique: true
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing or invalid asset definition name' });
  });

  it('Attempting to add an asset definition without an author should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/assets/definitions')
      .send({
        name: 'My asset definition',
        isContentPrivate: false,
        isContentUnique: true
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing or invalid asset definition author' });
  });

  it('Attempting to add an asset definition with an invalid index schema should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/assets/definitions')
      .send({
        name: 'My asset definition',
        author: '0x0000000000000000000000000000000000000001',
        isContentPrivate: false,
        isContentUnique: true,
        indexes: {}
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Indexes do not conform to index schema' });
  });

  it('Attempting to add an asset definition without indicating if the content should be private or not should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/assets/definitions')
      .send({
        name: 'My asset definition',
        author: '0x0000000000000000000000000000000000000001',
        isContentUnique: true
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing asset definition content privacy' });
  });

  it('Attempting to add an asset definition with an invalid description schema should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/assets/definitions')
      .send({
        name: 'My asset definition',
        author: '0x0000000000000000000000000000000000000001',
        descriptionSchema: 'INVALID',
        isContentPrivate: false,
        isContentUnique: true
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Invalid description schema' });
  });

  it('Attempting to add an asset definition with an invalid content schema should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/assets/definitions')
      .send({
        name: 'My asset definition',
        author: '0x0000000000000000000000000000000000000001',
        contentSchema: 'INVALID',
        isContentPrivate: false,
        isContentUnique: true
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Invalid content schema' });
  });

});

describe('Asset instances - argument validation', async () => {

  it('Attempting to add an asset instance without specifying the author should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/assets/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx')
      .send({
        assetDefinitionID: '',
        content: {}
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing or invalid asset instance author' });
  });

  it('Attempting to add an asset instance without specifying the content should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/assets/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx')
      .send({
        author: '0x0000000000000000000000000000000000000001'
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing or invalid asset content' });
  });

});
};
