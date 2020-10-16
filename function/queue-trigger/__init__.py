import json
import logging
import os
import tempfile
from typing import List

import azure.functions as func

from .main import (Args, BoundingBox, DBConfig, ProcessingParam, StorageConfig,
                   Tile, run)


def get_params_from_message(message: dict) -> List[ProcessingParam]:
    map_id = message['map_id']
    return [
        ProcessingParam(
            filename_postfix=map_id + '-' + m['postfix'],
            tile=Tile(
                x=m['tile']['x'],
                y=m['tile']['y'],
                z=m['tile']['z']
            ),
            bounds=BoundingBox(
                top_left=(
                    m['tl'][0],
                    m['tl'][1]
                ),
                bottom_right=(
                    m['br'][0],
                    m['br'][1]
                )
            )
        ) for m in message['coords']
    ]


def get_athlete_from_message(message: dict) -> int:
    return int(message['athlete_id'])


def get_args(athlete_id: int, params: List[ProcessingParam]) -> Args:
    return Args(
        tile_size_px=256,
        athlete_id=athlete_id,
        temp_dir=tempfile.mkdtemp(),
        processing_params=params,
        db_config=DBConfig(
            name=os.environ['DB_NAME'],
            user=os.environ['DB_USER'],
            host=os.environ['DB_HOST'],
            password=os.environ['DB_PASS']
        ),
        storage_config=StorageConfig(
            download_container_name=os.environ['STORAGE_CONTAINER_NAME'],
            upload_container_name=os.environ['UPLOAD_STORAGE_CONTAINER_NAME'],
            account_name=os.environ['STORAGE_ACCOUNT_NAME'],
            account_key=os.environ['STORAGE_ACCOUNT_KEY'],
            max_workers=int(os.environ['STORAGE_MAX_WORKERS'])
        )
    )


def main(msg: func.QueueMessage):
    logging.info('begin::queue-main')

    try:
        message = json.loads(msg.get_body().decode('utf-8'))

        args = get_args(
            get_athlete_from_message(message),
            get_params_from_message(message)
        )

        run(args)
    except Exception as e:
        logging.error('failed::queue-main')
        logging.error(e)
        raise e

    logging.info('end::queue-main')
