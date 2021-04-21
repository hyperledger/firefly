import { closeDown, setUp } from "./common";
import { ethereumTests } from "./ethereum";
describe('ethereum tests for initial bootstrap', async () => {
    before(async () => {
        await setUp('ethereum');
    });
    ethereumTests();
    after(async () => {
      await closeDown();
    });
});
