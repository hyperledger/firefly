import { closeDown, setUp } from "./common";
import { cordaTests } from "./corda";

describe('corda tests', async () => {
    before(async () => {
        await setUp('corda');
    });
    cordaTests();
    after(async () => {
       await closeDown();
    });
});