import { testMembersArgumentValidation } from "./argument-validation";

export const testMembers = async () => {
    describe('Member tests', async () => {
        testMembersArgumentValidation();
    });
};