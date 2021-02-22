import { testAssetArgumentValidation } from "./argument-validation";

export const testAssets = async () => {
    describe('Asset tests', async () => {
        testAssetArgumentValidation();
    });
};