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
  releaseTile(tile) {}
}

/**
* Initialize map
*/
function initMap() {
  const map = new google.maps.Map(document.getElementById("map"), {
      zoom: 13,
      maxZoom: 19,
      minZoom: 2,
      center: { // Austin, TX
          lat: 30.2729,
          lng: -97.7444
      },
      mapTypeId: 'terrain',
      mapTypeControlOptions: [],
  });
  map.setOptions({ styles: getMapStyle() });

  // Insert this overlay map type as the first overlay map type at
  // position 0. Note that all overlay map types appear on top of
  // their parent base map.
  map.overlayMapTypes.insertAt(
      0,
      new CoordMapType(new google.maps.Size(256, 256))
  );

  // ask for user location after map loads
  setTimeout(function(){ 
    if (navigator.geolocation) {
      navigator.geolocation.watchPosition(getPositionUpdateFunc(map), null, {});
    }
  }, 0);
}

function getPositionUpdateFunc(map) {
  return function({ coords: { latitude: lat, longitude: lng } }){
    map.setCenter({lat, lng})
    map.setZoom(15)
    map.panTo({lat, lng})
    if (typeof window.gpsMarker === 'undefined') {
        window.gpsMarker = new google.maps.Marker({
            position: {lat, lng},
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
        window.gpsMarker.setPosition({lat, lng})
    }
  }
}

/**
* Return map styles
*/
function getMapStyle() {
  return [
    {
      "elementType": "geometry",
      "stylers": [
        {
          "color": "#212121"
        }
      ]
    },
    {
      "elementType": "labels",
      "stylers": [
        {
          "visibility": "off"
        }
      ]
    },
    {
      "elementType": "labels.icon",
      "stylers": [
        {
          "visibility": "off"
        }
      ]
    },
    {
      "elementType": "labels.text.fill",
      "stylers": [
        {
          "color": "#757575"
        }
      ]
    },
    {
      "elementType": "labels.text.stroke",
      "stylers": [
        {
          "color": "#212121"
        }
      ]
    },
    {
      "featureType": "administrative",
      "elementType": "geometry",
      "stylers": [
        {
          "color": "#757575"
        },
        {
          "visibility": "off"
        }
      ]
    },
    {
      "featureType": "administrative.country",
      "elementType": "labels.text.fill",
      "stylers": [
        {
          "color": "#9e9e9e"
        }
      ]
    },
    {
      "featureType": "administrative.land_parcel",
      "stylers": [
        {
          "visibility": "off"
        }
      ]
    },
    {
      "featureType": "administrative.locality",
      "elementType": "labels.text.fill",
      "stylers": [
        {
          "color": "#bdbdbd"
        }
      ]
    },
    {
      "featureType": "administrative.neighborhood",
      "stylers": [
        {
          "visibility": "off"
        }
      ]
    },
    {
      "featureType": "poi",
      "stylers": [
        {
          "visibility": "off"
        }
      ]
    },
    {
      "featureType": "poi",
      "elementType": "labels.text.fill",
      "stylers": [
        {
          "color": "#757575"
        }
      ]
    },
    {
      "featureType": "poi.park",
      "elementType": "geometry",
      "stylers": [
        {
          "color": "#181818"
        }
      ]
    },
    {
      "featureType": "poi.park",
      "elementType": "labels.text.fill",
      "stylers": [
        {
          "color": "#616161"
        }
      ]
    },
    {
      "featureType": "poi.park",
      "elementType": "labels.text.stroke",
      "stylers": [
        {
          "color": "#1b1b1b"
        }
      ]
    },
    {
      "featureType": "road",
      "elementType": "geometry.fill",
      "stylers": [
        {
          "color": "#2c2c2c"
        }
      ]
    },
    {
      "featureType": "road",
      "elementType": "labels.icon",
      "stylers": [
        {
          "visibility": "off"
        }
      ]
    },
    {
      "featureType": "road",
      "elementType": "labels.text.fill",
      "stylers": [
        {
          "color": "#8a8a8a"
        }
      ]
    },
    {
      "featureType": "road.arterial",
      "elementType": "geometry",
      "stylers": [
        {
          "color": "#373737"
        }
      ]
    },
    {
      "featureType": "road.highway",
      "elementType": "geometry",
      "stylers": [
        {
          "color": "#3c3c3c"
        }
      ]
    },
    {
      "featureType": "road.highway.controlled_access",
      "elementType": "geometry",
      "stylers": [
        {
          "color": "#4e4e4e"
        }
      ]
    },
    {
      "featureType": "road.local",
      "elementType": "labels.text.fill",
      "stylers": [
        {
          "color": "#616161"
        }
      ]
    },
    {
      "featureType": "transit",
      "stylers": [
        {
          "visibility": "off"
        }
      ]
    },
    {
      "featureType": "transit",
      "elementType": "labels.text.fill",
      "stylers": [
        {
          "color": "#757575"
        }
      ]
    },
    {
      "featureType": "water",
      "elementType": "geometry",
      "stylers": [
        {
          "color": "#000000"
        }
      ]
    },
    {
      "featureType": "water",
      "elementType": "labels.text.fill",
      "stylers": [
        {
          "color": "#3d3d3d"
        }
      ]
    }
  ]
}