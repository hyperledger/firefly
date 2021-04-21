import { testAssets } from './assets';
import { testMembers } from './members';


export const cordaTests = async () => {
  testAssets();
  testMembers();
};