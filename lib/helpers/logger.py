"""Centralized logging for Konstellation"""

import logging
from logging import Handler
from logutils import colorize
from tqdm import tqdm


class TqdmLoggingHandler(Handler):
    def emit(self, record):
        try:
            msg = self.format(record)
            tqdm.write(msg)
            self.flush()
        except Exception:
            self.handleError(record)


class TqdmHandler(colorize.ColorizingStreamHandler):
    """A logging handler to cleanly handle logging with the progress bar."""
    def emit(self, record):
        try:
            message = self.format(record)
            stream = self.stream

            if not self.is_tty:
                stream.write(message)
            else:
                self.output_colorized(message)
            tqdm.write(getattr(self, 'terminator', '\n'))
            self.flush()
        except (KeyboardInterrupt, SystemExit) as e:
            raise e
        except Exception:
            self.handleError(record)

def get_logger(config) -> logging.Logger:
    logger = logging.getLogger('konstellation')
    if config.debug:
        logger.setLevel(logging.DEBUG)
    else:
        logger.setLevel(logging.INFO)

    handler = TqdmLoggingHandler()
    if config.debug:
        formatter = logging.Formatter(
            '[%(asctime)s][%(module)s][%(funcName)s] %(message)s',
            datefmt='%H:%M:%S')
    else:
        formatter = logging.Formatter(
            '[%(asctime)s] %(message)s', datefmt='%H:%M:%S')

    handler.setFormatter(formatter)
    logger.addHandler(handler)

    return logger
