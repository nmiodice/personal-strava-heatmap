import json
import logging
import os
import tempfile
import traceback 
from typing import List, Optional

import azure.functions as func

from .main import (Args, BoundingBox, DBConfig, ProcessingParam, StorageConfig,
                   Tile, run, get_db_conn)


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


def set_message_state(config: DBConfig, id: Optional[str], state: str) -> None:
    if not id:
        logging.info('set_message_state called with null ID')
        return
    logging.info('updating state of message {0} to {1}'.format(id, state))

    conn = None
    try:
        conn = get_db_conn(config)
        cur = conn.cursor()
        cur.execute(
            'UPDATE queueprocessingstate SET pstate = %s where message_id = %s;', (state, id))
        conn.commit()
        cur.close()
    finally:
        if conn:
            conn.close()


def main(msg: func.QueueMessage):
    logging.info('begin::queue-main')
    message_id = msg.id

    try:
        message = json.loads(msg.get_body().decode('utf-8'))

        args = get_args(
            get_athlete_from_message(message),
            get_params_from_message(message)
        )

        run(args)
        set_message_state(args.db_config, message_id, 'COMPLETE')
    except Exception as e:
        logging.error('failed::queue-main')
        logging.error(e)
        traceback.print_exc()
        set_message_state(args.db_config, message_id, 'FAILED')
        raise e

    logging.info('end::queue-main')
