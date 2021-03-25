import * as sinon from "sinon";
import io from 'socket.io-client';
import { StubbedInstance, stubInterface } from "ts-sinon";
import * as app2app from '../../../clients/app2app';

export const testApp2App = async () => {

describe('App2App', () => {

  let mocksSocketIo: StubbedInstance<SocketIOClient.Socket>;
  let callbacks: {[f: string]: Function};

  before(() => {
    callbacks = {};
    mocksSocketIo = stubInterface<SocketIOClient.Socket>();
    mocksSocketIo.on.callsFake((event, fn) => {
      callbacks[event] = fn;
      return mocksSocketIo;
    });
    sinon.stub(io, 'connect').returns(mocksSocketIo);
  });

  after(() => {
    (io.connect as sinon.SinonStub).restore();
  })

  beforeEach(() => {
    callbacks = {};
    app2app.reset();
  });

  describe('establishSocketIOConnection', () => {

    it('subscribes after connect', async () => {    
      sinon.assert.notCalled(mocksSocketIo.emit);
      callbacks['connect']();
      sinon.assert.calledWith(mocksSocketIo.emit, 'subscribe');
    });

    it('logs connect_error', async () => {    
      callbacks['connect_error'](new Error('pop'));
    });

    it('logs error', async () => {    
      callbacks['error'](new Error('pop'));
    });

    it('logs exception', async () => {    
      callbacks['exception'](new Error('pop'), {batch: 'test'});
    });

  });

});
};
