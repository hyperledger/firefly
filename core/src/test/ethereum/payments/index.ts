import { testPaymentArgumentValidation } from "./argument-validation";
import { testAuthored } from "./authored";
import { testAuthoredDescribed } from "./authored-described";
import { testUnauthored } from "./unauthored";
import { testUnauthoredDescribed } from "./unauthored-described";

export const testPayments = async () => {
    describe('Payment tests', async () => {
        testPaymentArgumentValidation();
        testAuthored();
        testAuthoredDescribed();
        testUnauthored();
        testUnauthoredDescribed();
    });
};