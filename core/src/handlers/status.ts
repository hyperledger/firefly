import * as apiGateway from '../clients/api-gateway';

export const handleGetStatusRequest = () => {
  return apiGateway.getStatus();
};