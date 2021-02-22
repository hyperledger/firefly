import { testMembersArgumentValidation } from "./argument-validation";
import { testMemberRegistration } from "./registration";

export const testMembers = async () => {
    describe('Member tests', async () => {
        testMembersArgumentValidation();
        testMemberRegistration();
    });
};