import { testAssetArgumentValidation } from "./argument-validation";
import { testSuite1011 } from "./authored/private/described-structured";
import { testSuite1010 } from "./authored/private/described-unstructured";
import { testSuite1001 } from "./authored/private/structured";
import { testSuite1000 } from "./authored/private/unstructured";
import { testSuite1111 } from "./authored/public/described-structured";
import { testSuite1110 } from "./authored/public/described-unstructured";
import { testSuite1101 } from "./authored/public/structured";
import { testSuite1100 } from "./authored/public/unstructured";
import { testSuite0011 } from "./unauthored/private/described-structured";
import { testSuite0010 } from "./unauthored/private/described-unstructured";
import { testSuite0001 } from "./unauthored/private/structured";
import { testSuite0000 } from "./unauthored/private/unstructured";
import { testSuite0111 } from "./unauthored/public/described-structured";
import { testSuite0110 } from "./unauthored/public/described-unstructured";
import { testSuite0101 } from "./unauthored/public/structured";
import { testSuite0100 } from "./unauthored/public/unstructured";

export const testAssets = async () => {
    describe('Asset tests', async () => {
        testAssetArgumentValidation();
        testSuite0001();
        testSuite0000();
        testSuite0010();
        testSuite0011();
        testSuite0100();
        testSuite0101();
        testSuite0110();
        testSuite0111();
        testSuite1000();
        testSuite1001();
        testSuite1010();
        testSuite1011();
        testSuite1100();
        testSuite1101();
        testSuite1110();
        testSuite1111();
    });
};