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
