// CONFIG INTERFACE

export interface IConfig {
  port: number
  apiGateway: {
    apiEndpoint: string
  }
  eventStreams: {
    wsEndpoint: string
    topic: string
  },
  ipfs: {
    apiEndpoint: string
    gatewayEndpoint: string
  },
  docExchange: {
    apiEndpoint: string
    socketIOEndpoint: string
  }
  appCredentials: {
    user: string
    password: string
  }  
}

export interface IStatus {
  totalAssetDefinitions: number
  totalPaymentDefinitions: number
}

// REQUEST INTERFACES

export interface IRequestMultiPartContent {
  author?: string
  assetDefinitionID?: string
  description?: Promise<string>
  contentStream: NodeJS.ReadableStream
  contentFileName: string
}

// EVENT STREAM INTERFACES

export interface IEventStreamMessage {
  address: string
  blockNumber: string
  transactionIndex: string
  transactionHash: string
  data: Object
  subId: string
  signature: string
  logIndex: string
}

export interface IEventMemberRegistered {
  member: string
  name: string
  app2appDestination: string
  docExchangeDestination: string
  timestamp: number
}

export interface IEventAssetDefinitionCreated {
  assetDefinitionID: string
  author: string
  name: string
  isContentPrivate: boolean
  isContentUnique: boolean
  contentSchemaHash?: string
  descriptionSchemaHash?: string
  timestamp: string
}

export interface IEventPaymentDefinitionCreated {
  paymentDefinitionID: string
  author: string
  name: string
  descriptionSchemaHash?: string
  timestamp: string
}

export interface IEventAssetInstanceCreated {
  assetInstanceID: string
  assetDefinitionID: string
  author: string
  descriptionHash?: string
  contentHash: string
  timestamp: string
}

export interface IEventPaymentInstanceCreated {
  paymentInstanceID: string
  paymentDefinitionID: string
  author: string
  recipient: string
  descriptionHash?: string
  amount: string
  timestamp: string
}

export interface IEventAssetInstancePropertySet {
  assetInstanceID: string
  author: string
  key: string
  value: string
  timestamp: string
}

// DATABASE INTERFACES

export interface IDBBlockchainData {
  blockNumber: number,
  transactionHash: string
}

export interface IDBMember {
  _id?: string
  address: string
  app2appDestination: string
  docExchangeDestination: string
  timestamp: number
  confirmed: boolean
  blockchainData?: IDBBlockchainData
  owned: boolean
}

export interface IDBAssetDefinition {
  _id?: string
  assetDefinitionID: string
  author: string
  name: string
  isContentPrivate: boolean
  isContentUnique: boolean
  descriptionSchemaHash?: string
  descriptionSchema?: object
  contentSchemaHash?: string
  contentSchema?: object
  timestamp: number
  confirmed: boolean
  blockchainData?: IDBBlockchainData
  conflict?: boolean
}

export interface IDBPaymentDefinition {
  _id?: string
  paymentDefinitionID: string
  author: string
  name: string
  descriptionSchema?: object
  descriptionSchemaHash?: string
  timestamp: number
  confirmed: boolean
  blockchainData?: IDBBlockchainData
}

export interface IDBAssetInstance {
  _id?: string
  assetInstanceID: string
  assetDefinitionID: number
  author: string
  descriptionHash?: string
  description?: string
  content?: string
  contentHash?: string
  confirmed: boolean
  conflict: boolean
  blockchainData?: IDBBlockchainData
  timestamp: number
  properties: {
    [author: string]: {
      [key: string]: {
        value: string
        timestamp: number
        confirmed: boolean
        blockchainData?: IDBBlockchainData
      } | undefined
    } | undefined
  }
}

export interface IDBPaymentInstance {
  _id?: string
  paymentInstanceID: string
  paymentDefinitionID: string
  author: string
  recipient: string
  amount: number
  descriptionHash?: string
  description: string
  confirmed: boolean
  blockchainData?: IDBBlockchainData
  timestamp: number
}

// DOCUMENT EXCHANGE INTERFACES

export interface IDocExchangeTransferData {
  transferId: string
  transferHash: string
  hash: string
  from: string
  to: string
  senderSignature: string
  recipientSignature: string
  document: string
  timestamp: string
  status: 'sent' | 'received' | 'failed'
}

export interface IDocExchangeListener {
  (transferData: IDocExchangeTransferData): void
}