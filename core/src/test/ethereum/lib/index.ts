import { testBatchManager } from "./batch-manager-test";
import { testBatchProcessor } from "./batch-processor-test";
import { testUtils } from "./config";
import { testSettings } from "./settings";
import { testLogging } from "./logging";

export const testLibraryFunctions = () => {
    describe('Lib tests', async () => {
        testBatchManager();
        testBatchProcessor();
        testUtils();
        testSettings();
        testLogging();
    });
};