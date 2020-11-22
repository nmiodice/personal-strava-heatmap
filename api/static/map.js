
function resetWindowParams() {
  window.params = new URLSearchParams(window.location.search);
}

/**
 * Adds tile image overlay when pannign around the map
 */
class CoordMapType {
  constructor(tileSize) {
    this.tileSize = tileSize;
  }
  getTile(coord, zoom, ownerDocument) {
    const img = ownerDocument.createElement("img");
    const endpoint = $('#tile_endpoint')[0].value
    const map_id = $('#map_id')[0].value
    const tile_name = '-' + coord.x + '-' + coord.y + '-' + zoom + '.png'

    img.onerror = "this.style.display='none';"
    img.alt = ""
    img.src = endpoint + map_id + tile_name
    return img
  }
  releaseTile(tile) { }
}

/**
* Initialize map
*/
function initMap() {
  resetWindowParams()
  configureWindowMap()
  configureMapListeners()
  configureButtomListener()
  applyMapOverlay()
  triggerGPSEnablement()
}

function configureButtomListener() {
  $('#location_button').click(function () {
    if (navigator.geolocation) {
      navigator.geolocation.getCurrentPosition(function ({ coords: { latitude: lat, longitude: lng } }) {
        map.setCenter({ lat, lng })
        map.panTo({ lat, lng })
        map.setZoom(13)
      }, function () {
        alert('You must grant access to your location in order to use this feature.')
      });
    }
  });
}

function getQueryParam(paramName, defaultValue) {
  val = window.params.get(paramName)
  if (val != null) {
    return (defaultValue.constructor)(val)
  }

  return defaultValue
}

function configureWindowMap() {
  window.map = new google.maps.Map(document.getElementById("map"), {
    zoom: getQueryParam('z', 13),
    maxZoom: 19,
    minZoom: 2,
    center: { // Default is Austin, TX
      lat: getQueryParam('lat', 30.2729),
      lng: getQueryParam('lon', -97.7444)
    },
    mapTypeId: 'terrain',
    mapTypeControlOptions: [],
  });
  window.map.setOptions({
    styles: getMapStyle(),
    zoomControl: true,
    scaleControl: true,
    mapTypeControl: false,
    streetViewControl: false,
    rotateControl: false,
    fullscreenControl: false
  });
}

function configureMapListeners() {
  let positionListener = function () {
    newParams = new URLSearchParams(window.params)
    newParams.set('lat', window.map.getCenter().lat())
    newParams.set('lon', window.map.getCenter().lng())
    newParams.set('z', window.map.getZoom())

    history.replaceState(null, null, "?" + newParams.toString());
    resetWindowParams()
  }

  window.map.addListener("center_changed", positionListener);
  window.map.addListener("zoom_changed", positionListener);
}

function applyMapOverlay() {
  // Insert this overlay map type as the first overlay map type at
  // position 0. Note that all overlay map types appear on top of
  // their parent base map.
  window.map.overlayMapTypes.insertAt(
    0,
    new CoordMapType(new google.maps.Size(256, 256))
  );
}

function triggerGPSEnablement(map) {
  setTimeout(function () {
    if (navigator.geolocation) {
      navigator.geolocation.getCurrentPosition(setPositionMarker);
    }
  }, 0);
}

function setPositionMarker({ coords: { latitude: lat, longitude: lng } }) {
  if (typeof window.gpsMarker === 'undefined') {
    window.gpsMarker = new google.maps.Marker({
      position: { lat, lng },
      map: map,
      icon: {
        path: google.maps.SymbolPath.CIRCLE,
        scale: 8,
        fillOpacity: 1,
        strokeWeight: 2,
        fillColor: '#5384ED',
        strokeColor: '#ffffff',
      },
    });
  } else {
    window.gpsMarker.setPosition({ lat, lng })
  }
}
