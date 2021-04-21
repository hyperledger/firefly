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
        author: 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US',
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

  it('Attempting to add an asset definition without an valid author name should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/assets/definitions')
      .send({
        name: 'My asset definition',
        isContentPrivate: false,
        isContentUnique: true,
        author: "invalid name"
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing or invalid asset definition author' });
  });

  it('Attempting to add an asset definition without indicating if the content should be private or not should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/assets/definitions')
      .send({
        name: 'My asset definition',
        author: 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US',
        isContentUnique: true
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing asset definition content privacy' });
  });

  it('Attempting to add an asset definition without a definition id should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/assets/definitions')
      .send({
        name: 'My asset definition',
        author: 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US',
        isContentUnique: true,
        isContentPrivate: false
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing asset definition id' });
  });

  it('Attempting to add an asset definition with an invalid index schema should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/assets/definitions')
      .send({
        name: 'My asset definition',
        author: 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US',
        assetDefinitionID: 'some-uuid',
        isContentPrivate: false,
        isContentUnique: true,
        indexes: {}
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Indexes do not conform to index schema' });
  });

  it('Attempting to add an asset definition with an invalid description schema should raise an error', async () => {
    const result = await request(app)
      .post('/api/v1/assets/definitions')
      .send({
        name: 'My asset definition',
        author: 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US',
        assetDefinitionID: 'some-uuid',
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
        author: 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US',
        assetDefinitionID: 'some-uuid',
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
        author: 'CN=Node of node1 for env1, O=Kaleido, L=Raleigh, C=US'
      })
      .expect(400);
    assert.deepStrictEqual(result.body, { error: 'Missing or invalid asset content' });
  });
});
};