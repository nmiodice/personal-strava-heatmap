/*
 * This demo illustrates the coordinate system used to display map tiles in the
 * API.
 *
 * Tiles in Google Maps are numbered from the same origin as that for
 * pixels. For Google's implementation of the Mercator projection, the origin
 * tile is always at the northwest corner of the map, with x values increasing
 * from west to east and y values increasing from north to south.
 *
 * Try panning and zooming the map to see how the coordinates change.
 */
class CoordMapType {
    constructor(tileSize) {
      this.tileSize = tileSize;
    }
    getTile(coord, zoom, ownerDocument) {
      const img = ownerDocument.createElement("img");
      const endpoint = $( '#tile_endpoint' )[0].value
      const map_id = $( '#map_id' )[0].value
      const tile_name = '-' + coord.x + '-' + coord.y + '-' + zoom + '.png'

      img.onerror="this.style.display='none';"
      img.src = endpoint + map_id + tile_name
      return img
    }
    releaseTile(tile) {}
  }
  
  function initMap() {
    const map = new google.maps.Map(document.getElementById("map"), {
      zoom: 17,
      center: { lat: 30.2672, lng: -97.7431 },
    });
    // Insert this overlay map type as the first overlay map type at
    // position 0. Note that all overlay map types appear on top of
    // their parent base map.
    map.overlayMapTypes.insertAt(
      0,
      new CoordMapType(new google.maps.Size(256, 256))
    );
  }