
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
    },
    ipfsSha256:'0x2ef1f576d187bb132068095b05a6796891bd0ee1bd69037c6c60f2a6b705d35a',
    ipfsMultiHash: 'QmRVuTF2ktoKop95VbgARPZksdfZgpxx6xCJwFE3UbcMmT'
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
  },
  sample: {
    object: {
      my_content_string: 'sample content string',
      my_content_number: 456,
      my_content_boolean: true
    },
    ipfsSha256: '0x12e850feabadae5158666a3d03b449fbd4f04582ef0c9b5a91247a02af110016',
    ipfsMultiHash: 'QmPcTWXWiUEwect513QdDtw1wa9QWcRgGTVebGbjhMKNxV',
    docExchangeSha256: '0xb681804d7f63b532394091f9a0eab0c94e82a92332b4d299ba7493903b27c9e1'
  }
};

export const testIndexes = [
  {
    fields: ["author"],
    unique: false
  },
  {
    fields: ["author", "assetDefinitionID"],
    unique: false
  }
];
