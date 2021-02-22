import { testAssetArgumentValidation } from "./argument-validation";
import { testAssetsAuthoredPrivateStructured } from "./authored-private-structured";

export const testAssets = async () => {
    describe('Asset tests', async () => {
        testAssetArgumentValidation();
        testAssetsAuthoredPrivateStructured();
    });
};