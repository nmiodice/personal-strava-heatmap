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

    const marker = new google.maps.Marker({
      position: {lat, lng},
      map: map,
      icon: {
        path: google.maps.SymbolPath.CIRCLE,
        scale: 6,
        fillOpacity: 1,
        strokeWeight: 2,
        fillColor: '#5384ED',
        strokeColor: '#ffffff',
      },
    });
  }
}

/**
* Return map styles
*/
function getMapStyle() {
  return [{
          "elementType": "geometry",
          "stylers": [{
              "color": "#242f3e"
          }]
      },
      {
          "elementType": "labels",
          "stylers": [{
              "visibility": "off"
          }]
      },
      {
          "elementType": "labels.text.fill",
          "stylers": [{
              "color": "#746855"
          }]
      },
      {
          "elementType": "labels.text.stroke",
          "stylers": [{
              "color": "#242f3e"
          }]
      },
      {
          "featureType": "administrative.land_parcel",
          "stylers": [{
              "visibility": "off"
          }]
      },
      {
          "featureType": "administrative.locality",
          "elementType": "labels.text.fill",
          "stylers": [{
              "color": "#d59563"
          }]
      },
      {
          "featureType": "administrative.neighborhood",
          "stylers": [{
              "visibility": "off"
          }]
      },
      {
          "featureType": "poi",
          "elementType": "labels.text.fill",
          "stylers": [{
              "color": "#d59563"
          }]
      },
      {
          "featureType": "poi.park",
          "elementType": "geometry",
          "stylers": [{
              "color": "#263c3f"
          }]
      },
      {
          "featureType": "poi.park",
          "elementType": "labels.text.fill",
          "stylers": [{
              "color": "#6b9a76"
          }]
      },
      {
          "featureType": "road",
          "elementType": "geometry",
          "stylers": [{
              "color": "#38414e"
          }]
      },
      {
          "featureType": "road",
          "elementType": "geometry.stroke",
          "stylers": [{
              "color": "#212a37"
          }]
      },
      {
          "featureType": "road",
          "elementType": "labels.text.fill",
          "stylers": [{
              "color": "#9ca5b3"
          }]
      },
      {
          "featureType": "road.highway",
          "elementType": "geometry",
          "stylers": [{
              "color": "#746855"
          }]
      },
      {
          "featureType": "road.highway",
          "elementType": "geometry.stroke",
          "stylers": [{
              "color": "#1f2835"
          }]
      },
      {
          "featureType": "road.highway",
          "elementType": "labels.text.fill",
          "stylers": [{
              "color": "#f3d19c"
          }]
      },
      {
          "featureType": "transit",
          "elementType": "geometry",
          "stylers": [{
              "color": "#2f3948"
          }]
      },
      {
          "featureType": "transit.station",
          "elementType": "labels.text.fill",
          "stylers": [{
              "color": "#d59563"
          }]
      },
      {
          "featureType": "water",
          "elementType": "geometry",
          "stylers": [{
              "color": "#17263c"
          }]
      },
      {
          "featureType": "water",
          "elementType": "labels.text.fill",
          "stylers": [{
              "color": "#515c6d"
          }]
      },
      {
          "featureType": "water",
          "elementType": "labels.text.stroke",
          "stylers": [{
              "color": "#17263c"
          }]
      }
  ]
}