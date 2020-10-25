export const constants = {
  DATA_DIRECTORY: process.env.DATA_DIRECTORY || '/data',
  CONFIG_FILE_NAME: 'config.json',
  DEFAULT_PAGINATION_LIMIT: 100,
  MEMBERS_DATABASE_FILE_NAME: 'members.json'
};

export const contractEventSignatures = {
  MEMBER_REGISTERED: 'MemberRegistered(address,string,string,string,uint256)'
};

export const getTimestamp = () => {
  return Math.round(new Date().getTime() / 1000);
};