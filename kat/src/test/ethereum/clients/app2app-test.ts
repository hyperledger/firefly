// Copyright © 2021 Kaleido, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
