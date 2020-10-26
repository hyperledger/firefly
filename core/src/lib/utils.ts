export const constants = {
  DATA_DIRECTORY: process.env.DATA_DIRECTORY || '/data',
  CONFIG_FILE_NAME: 'config.json',
  IPFS_TIMEOUT_MS: 15000,
  DEFAULT_PAGINATION_LIMIT: 100,
  MEMBERS_DATABASE_FILE_NAME: 'members.json',
  EVENT_STREAM_WEBSOCKET_RECONNECTION_DELAY_SECONDS: 5
};

export const contractEventSignatures = {
  MEMBER_REGISTERED: 'MemberRegistered(address,string,string,string,uint256)'
};

export const getTimestamp = () => {
  return Math.round(new Date().getTime() / 1000);
};