export const testDescription = {
  schema: {
    object: {
      type: 'object',
      required: ['my_description_string', 'my_description_number', 'my_description_boolean'],
      properties: {
        my_description_string: {
          type: 'string'
        },
        my_description_number: {
          type: 'number'
        },
        my_description_boolean: {
          type: 'boolean'
        }
      }
    },
    ipfsSha256: '0xbde03e0e77b5422ff3ce4889752ac9450343420a6a4354542b9fd14fd5fa435c',
    ipfsMultiHash: 'Qmb7r5v11TYsJE8dYfnBFjwQsKapX1my7hzfAnd5GFq2io',
  },
  sample: {
    object: {
      my_description_string: 'sample description string',
      my_description_number: 123,
      my_description_boolean: true
    }
  }
};

export const testContent = {
  schema: {
    object: {
      type: 'object',
      required: ['my_content_string', 'my_content_number', 'my_content_boolean'],
      properties: {
        my_content_string: {
          type: 'string'
        },
        my_content_number: {
          type: 'number'
        },
        my_content_boolean: {
          type: 'boolean'
        }
      }
    },
    ipfsSha256: '0x64c97929fb90da1b94d560a29d8522c77b6c662588abb6ad23f1a0377250a2b0',
    ipfsMultiHash: 'QmV85fRf9jng5zhcSC4Zef2dy8ypouazgckRz4GhA5cUgw'
  },
  sample: {
    object: {
      my_content_string: 'sample content string',
      my_content_number: 456,
      my_content_boolean: true
    }
  }
};