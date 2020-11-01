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
  totalAssetDefinitions: number,
  totalAssetInstances: number,
  totalPaymentDefinitionsc: number,
  totalPaymentInstances: number
}

// REQUEST INTERFACES

export interface IRequestMultiPartContent {
  author?: string
  assetType?: number
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
  contentSchemaHash?: string
  descriptionSchemaHash?: string
  timestamp: string
}


// DATABASE INTERFACES

export type TAssetStatus = 'authored' | 'available' | 'retrieved'

export interface IDBMember {
  _id?: string
  address: string
  app2appDestination: string
  docExchangeDestination: string
  timestamp: number
  confirmed: boolean
  owned: boolean
}

export interface IDBAssetDefinition {
  _id?: string
  assetDefinitionID?: number
  author: string
  name: string
  isContentPrivate: boolean
  descriptionSchema?: object
  contentSchema?: object
  timestamp: number
}

export interface IDBAssetInstance {
  _id?: string
  assetInstanceID?: number
  author: string
  description?: string
  content?: string
  contentHash?: string
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