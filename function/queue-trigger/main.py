import concurrent
import io
import json
import logging
import math
import os
import tempfile
import threading
from collections import defaultdict, namedtuple
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass
from typing import Any, Dict, List, Optional, Sequence, Set, Tuple

import numpy as np
import PIL
import psycopg2
from azure.storage.blob import BlobServiceClient, ContentSettings
from PIL import Image, ImageFilter
from scipy.ndimage.filters import gaussian_filter


@dataclass
class CoordinateSummary:
    points: Set[Tuple[int, int]]


@dataclass(unsafe_hash=True)
class BoundingBox:
    top_left: Tuple[float, float]
    bottom_right: Tuple[float, float]


@dataclass(unsafe_hash=True)
class Tile:
    x: int
    y: int
    z: int


@dataclass(unsafe_hash=True)
class ProcessingParam:
    filename_postfix: str
    bounds: BoundingBox
    tile: Tile


@dataclass
class DBConfig:
    name: str
    host: str
    user: str
    password: str


@dataclass
class StorageConfig:
    download_container_name: str
    upload_container_name: str
    account_name: str
    account_key: str
    max_workers: int


@dataclass
class Args:
    tile_size_px: int
    athlete_id: int
    temp_dir: str
    processing_params: List[ProcessingParam]
    db_config: DBConfig
    storage_config: StorageConfig


@dataclass
class ActivityRef:
    data_ref: str


JsonDict = Dict[str, Any]

ACTIVITIES_DOWNLOAD_LOCK: threading.Lock = threading.Lock()
ACTIVITIES_AS_NUMPY_WORLD_COORDS: Dict[int, List[np.ndarray]] = dict()


def get_processing_params() -> List[ProcessingParam]:
    return [
        ProcessingParam('foo-1.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                 bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('foo-2.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                 bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('foo-3.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                 bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('foo-4.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                 bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('foo-5.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                 bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('foo-6.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                 bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('foo-7.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                 bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('foo-8.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                 bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('foo-9.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                 bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('foo-10.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                  bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('bar-1.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                 bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('bar-2.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                 bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('bar-3.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                 bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('bar-4.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                 bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('bar-5.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                 bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('bar-6.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                 bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('bar-7.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                 bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('bar-8.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                 bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('bar-9.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                 bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
        ProcessingParam('bar-10.png', BoundingBox(top_left=(42.47918714391391, -71.41422271728516),
                                                  bottom_right=(42.478933935077755, -71.41387939453125)), Tile(316279, 387364, 20)),
    ]


def get_args() -> Args:
    return Args(
        tile_size_px=256,
        athlete_id=32401715,
        temp_dir=tempfile.mkdtemp(),
        processing_params=get_processing_params(),
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


def get_activity_refs(athlete_id: int, config: DBConfig) -> List[ActivityRef]:
    refs = []
    conn = None
    try:
        conn = psycopg2.connect(
            host=config.host,
            database=config.name,
            user=config.user + "@" + config.host,
            password=config.password)

        cur = conn.cursor()
        cur.execute(
            'SELECT activity_data_ref FROM stravaactivity WHERE athlete_id = %s', (athlete_id,))
        row = cur.fetchone()

        while row is not None and len(row) and row[0] is not None:
            refs.append(ActivityRef(row[0]))
            row = cur.fetchone()

        cur.close()
    finally:
        if conn:
            conn.close()

    return refs


def download_activities(refs: List[ActivityRef], config: StorageConfig) -> List[JsonDict]:
    json_docs = []
    service_client = BlobServiceClient(
        account_url="https://{0}.blob.core.windows.net".format(
            config.account_name),
        credential=config.account_key)

    jobs = []
    with ThreadPoolExecutor(max_workers=config.max_workers) as executor:
        for ref in refs:
            blob_client = service_client.get_blob_client(
                config.download_container_name, ref.data_ref)

            def __dl_func(client):
                js_string = client.download_blob().readall()
                json_docs.append(json.loads(js_string))
                downloaded = len(json_docs)
                if downloaded % 20 == 0:
                    logging.info(
                        'downloaded {}/{} activities'.format(len(json_docs), len(refs)))

            jobs.append(executor.submit(__dl_func, blob_client))

    concurrent.futures.as_completed(jobs)
    logging.info(
        'downloaded {}/{} activities'.format(len(json_docs), len(refs)))
    return json_docs


def upload_images(images: Dict[ProcessingParam, PIL.Image.Image], config: StorageConfig) -> None:

    service_client = BlobServiceClient(
        account_url="https://{0}.blob.core.windows.net".format(
            config.account_name),
        credential=config.account_key)

    jobs = []
    with ThreadPoolExecutor(max_workers=config.max_workers) as executor:
        for param in images.keys():
            # convert image to bytes and upload to blob
            def __upload_func(theParam):
                blob_client = service_client.get_blob_client(
                    config.upload_container_name, theParam.filename_postfix)

                byte_array = io.BytesIO()
                images[theParam].save(byte_array, format='PNG')

                blob_client.upload_blob(byte_array.getvalue(), overwrite=True)
                blob_client.set_http_headers(ContentSettings(cache_control="max-age=60"))

            jobs.append(executor.submit(__upload_func, param))

    concurrent.futures.as_completed(jobs)


def measurement_to_percent(measurement, min_val, max_val):
    return float(measurement - min_val)/float(max_val-min_val)


def project_to_world_coordinates(tile_size_px: int, coords: np.ndarray) -> np.ndarray:
    xs = tile_size_px * (0.5 + coords[:, 1] / 360)
    siny = np.clip(
        np.sin(coords[:, 0] * math.pi / 180.0), a_min=-0.9999, a_max=0.9999)
    ys = tile_size_px * (0.5 - np.log((1 + siny) / (1 - siny)) / (4 * math.pi))

    return np.vstack([xs, ys]).T


def world_coord_to_pxs(coords: np.ndarray, scale: int) -> np.ndarray:
    return (coords * scale).astype(int)


def lat_lon_to_pxs(tile_size_px: int, coords: np.ndarray, scale: int) -> np.ndarray:
    return world_coord_to_pxs(
        project_to_world_coordinates(tile_size_px, coords),
        scale
    )


def add_to_coordinate_summary(coords: np.ndarray, tile_size_px: int, param: ProcessingParam, summary: CoordinateSummary):

    bounds = param.bounds
    scale = 1 << param.tile.z

    bounds_px = lat_lon_to_pxs(
        tile_size_px,
        np.array([bounds.top_left, bounds.bottom_right]),
        scale
    )
    top_left_px = bounds_px[0]
    bottom_right_px = bounds_px[1]

    coords_as_px = world_coord_to_pxs(coords, scale)

    # most activities won't be in the bounds. filter these out now so that
    # the interpolation process below will always be interpolating (and therefore
    # slowing the program down) over in-bound data
    if filter_in_bounds(coords_as_px, top_left_px, bottom_right_px).size == 0:
        return

    coordTimestamps = range(len(coords_as_px))
    coordFineGrainedTimestamps = np.arange(0, len(coords_as_px), .1)

    interpLats = np.interp(coordFineGrainedTimestamps,
                           coordTimestamps, coords_as_px[:, 0])
    interpLons = np.interp(coordFineGrainedTimestamps,
                           coordTimestamps, coords_as_px[:, 1])
    interpCoords = np.vstack([interpLats, interpLons]).T

    filteredInterpCoordPx = filter_in_bounds(
        interpCoords, top_left_px, bottom_right_px)
    for px in filteredInterpCoordPx:
        summary.points.add(adjust_pixel_for_tile(px, top_left_px, tile_size_px))


def filter_in_bounds(coords: np.ndarray, top_left_px: Tuple[int, int], bottom_right_px: Tuple[int, int]) -> np.ndarray:
    return coords[(
        (coords[:, 0] > top_left_px[0]) &
        (coords[:, 0] < bottom_right_px[0]) &
        (coords[:, 1] > top_left_px[1]) &
        (coords[:, 1] < bottom_right_px[1])
    )]


def adjust_pixel_for_tile(px: Tuple[float, float], top_left_px: Tuple[int, int], tile_size_px: int) -> Tuple[int, int]:
    return (
        min(tile_size_px - 1, int(px[0] - top_left_px[0])),
        min(tile_size_px - 1, int(px[1] - top_left_px[1])),
    )


def compute_coordinates_summaries(np_world_coords: List[np.ndarray], tile_size_px: int, params: List[ProcessingParam]):
    summaries = {
        param: CoordinateSummary(set())
        for param in params
    }

    for coords in np_world_coords:
        for param, summary in summaries.items():
            add_to_coordinate_summary(coords, tile_size_px, param, summary)

    return summaries


def process_coordinate_summary(tile_size_px: int, coord_summary: CoordinateSummary) -> PIL.Image.Image:
    imageMap = np.zeros((tile_size_px, tile_size_px, 4), dtype=np.uint8)

    # axis 0 is Y, axis 1 is X
    for point in coord_summary.points:
        imageMap[point[1], point[0]] = [255, 0, 0, 255]

    blurredImageMap = gaussian_filter(imageMap, sigma=(.8, .8, .8))
    maxPxVal = np.max(blurredImageMap)
    blurredImageMap = blurredImageMap * (255.0 / maxPxVal)
    blurredImageMap = blurredImageMap.astype(np.uint8, copy=False)

    return Image.fromarray(blurredImageMap, 'RGBA')


def parse_activity_coordinates(docs: List[JsonDict]) -> np.ndarray:
    # filter out documents with errors
    docsWithCoords = [d for d in docs if 'errors' not in d]
    logging.warning('filtered {0} of {1} docs'.format(
        len(docs) - len(docsWithCoords), len(docs)))

    # convert coordinates for all documents
    numpyCoords = []
    for doc in docsWithCoords:
        for stream in doc:
            if 'type' in stream and stream['type'] == 'latlng':  # type: ignore
                numpyCoords.append(stream['data'])    # type: ignore

    return np.array(numpyCoords, dtype=object)


def get_activities_as_numpy(args: Args) -> List[np.ndarray]:
    global ACTIVITIES_AS_NUMPY_WORLD_COORDS

    with ACTIVITIES_DOWNLOAD_LOCK:
        if args.athlete_id not in ACTIVITIES_AS_NUMPY_WORLD_COORDS:
            logging.info('begin::get_activity_refs')
            activity_refs = get_activity_refs(args.athlete_id, args.db_config)
            logging.info('end::get_activity_refs')

            logging.info('begin::download_activities')
            activity_json_docs = download_activities(
                activity_refs, args.storage_config)
            logging.info('end::download_activities')

            logging.info('begin::activity_to_world_coordinates')
            coordinates = parse_activity_coordinates(activity_json_docs)

            ACTIVITIES_AS_NUMPY_WORLD_COORDS[args.athlete_id] = [
                project_to_world_coordinates(
                    args.tile_size_px, np.array(coords))
                for coords in coordinates
            ]
            logging.info('end::activity_to_world_coordinates')
        else:
            logging.info('using cached ride data')

        return ACTIVITIES_AS_NUMPY_WORLD_COORDS[args.athlete_id]


def run(args: Args):
    configure_logger()

    logging.info('begin::run')
    logging.info('\tprocessing_params: ' + str(len(args.processing_params)))
    logging.info('\tathlete_id: ' + str(args.athlete_id))
    numpy_world_coords = get_activities_as_numpy(args)
    logging.info('\tdocument_count: ' + str(len(numpy_world_coords)))

    logging.info('begin::compute_coordinates_summaries')
    coord_summaries = compute_coordinates_summaries(
        numpy_world_coords, args.tile_size_px, args.processing_params)
    logging.info('end::compute_coordinates_summaries')

    logging.info('begin::process_coordinate_summary')
    images = {
        param: process_coordinate_summary(args.tile_size_px, coord_summary)
        for param, coord_summary
        in coord_summaries.items()
    }
    logging.info('end::process_coordinate_summary')

    logging.info('begin::upload_images')
    upload_images(images, args.storage_config)
    logging.info('end::upload_images')

    logging.info('end::run')

def configure_logger():
    logging.basicConfig(
        level=logging.INFO,
    )
    logging.getLogger('azure.core').setLevel(logging.WARN)
    logging.getLogger('urllib3.connectionpool').setLevel(logging.ERROR)

if __name__ == '__main__':
    run(get_args())
