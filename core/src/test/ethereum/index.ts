import { testLibraryFunctions } from './lib';
import { testMembers } from './members';
import { testPayments } from './payments';
import { testAssets } from './assets';
export const ethereumTests = async () => {
// test Assets
testAssets();

testLibraryFunctions();

testMembers();

testPayments();

};
