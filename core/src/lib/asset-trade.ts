import { IApp2AppMessage } from "./interfaces";

export const assetTradeHandler = (message: IApp2AppMessage) => {
    console.log(message.headers);
    console.log('AAAA')
};