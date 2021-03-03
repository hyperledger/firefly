import { testAssetArgumentValidation } from "./argument-validation";
import { testAuthoredPrivateDescribedStructured } from "./authored/private/described-structured";
import { testAuthoredPrivateDescribedUnstructured } from "./authored/private/described-unstructured";
import { testAuthoredPrivateStructured } from "./authored/private/structured";
import { testAuthoredPrivateUnstructured } from "./authored/private/unstructured";
import { testAuthoredPublicDescribedStructured } from "./authored/public/described-structured";
import { testAuthoredPublicDescribedUnstructured } from "./authored/public/described-unstructured";
import { testAuthoredPublicStructured } from "./authored/public/structured";
import { testAuthoredPublicUnstructured } from "./authored/public/unstructured";
import { testUnauthoredPrivateDescribedStructured } from "./unauthored/private/described-structured";
import { testUnauthoredPrivateDescribedUnstructured } from "./unauthored/private/described-unstructured";
import { testUnauthoredPrivateStructured } from "./unauthored/private/structured";
import { testUnauthoredPrivateUnstructured } from "./unauthored/private/unstructured";
import { testUnauthoredPublicDescribedStructured } from "./unauthored/public/described-structured";
import { testUnauthoredPublicDescribedUnstructured } from "./unauthored/public/described-unstructured";
import { testUnauthoredPublicStructured } from "./unauthored/public/structured";
import { testUnauthoredPublicUnstructured } from "./unauthored/public/unstructured";

export const testAssets = async () => {
    describe('Asset tests', async () => {
        testAssetArgumentValidation();
        testAuthoredPrivateDescribedStructured();
        testAuthoredPrivateDescribedUnstructured();
        testAuthoredPrivateStructured();
        testAuthoredPrivateUnstructured();
        testAuthoredPublicDescribedStructured();
        testAuthoredPublicDescribedUnstructured();
        testAuthoredPublicStructured();
        testAuthoredPublicUnstructured();
        testUnauthoredPrivateDescribedStructured();
        testUnauthoredPrivateDescribedUnstructured();
        testUnauthoredPrivateStructured();
        testUnauthoredPrivateUnstructured();
        testUnauthoredPublicDescribedStructured();
        testUnauthoredPublicDescribedUnstructured();
        testUnauthoredPublicStructured();
        testUnauthoredPublicUnstructured();
    });
};