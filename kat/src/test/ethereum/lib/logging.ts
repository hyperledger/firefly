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

import { Logger, setLogLevel } from '../../../lib/logging';

export const testLogging = async () => {

  describe('Logging', () => {

    it('logs all the levels', () => {

      setLogLevel('trace')

      const logger = new Logger('unit test');
      logger.error('this is an error message');
      logger.warn('this is an warning message');
      logger.warn('this is an info message');
      logger.debug('this is a debug message');
      logger.trace('this is a trace message');

      // Log empty
      logger.info();
      
      // Log a dodgy axios message exercising paths
      logger.info({
        isAxiosError: true,
      })

      // Log an axios error that looks like a stream
      logger.info({
        isAxiosError: true,
        response: {
          data: {
            on: () => null,
          },  
        },
      })
      
    });

  });
};
