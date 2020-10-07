// CONFIG INTERFACE

export interface IConfig {
  ethconnect: {
    wsURL: string
    credentials: {
      user: string
      password: string
    }    
  }
}

