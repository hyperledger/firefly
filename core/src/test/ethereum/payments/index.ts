import { testPaymentArgumentValidation } from "./argument-validation";
import { testPaymentDefinitions10 } from "./authored";
import { testPaymentDefinitions11 } from "./authored-described";
import { testPaymentDefinitions00 } from "./unauthored";
import { testPaymentDefinitions01 } from "./unauthored-described";

export const testPayments = async () => {
    describe('Payment tests', async () => {
        testPaymentArgumentValidation();
        testPaymentDefinitions00();
        testPaymentDefinitions01();
        testPaymentDefinitions10();
        testPaymentDefinitions11();
    });
};