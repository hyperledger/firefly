import { testApp2App } from "./app2app-test";

export const testClients = async () => {
    describe('Client tests', async () => {
        testApp2App();
    });
};