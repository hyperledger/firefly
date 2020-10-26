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

export interface IMemberRegisteredEvent {
  member: string
  name: string
  app2appDestination: string
  docExchangeDestination: string
  timestamp: number
}

// DATABASE INTERFACES

export interface IDBMember {
  _id?: string
  address: string
  app2appDestination: string
  docExchangeDestination: string
  timestamp: number
  confirmed: boolean
  owned: boolean
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