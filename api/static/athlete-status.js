$( document ).ready(function() {
    window.refreshTimer = setInterval(refreshStatus, 5000)
    refreshStatus()
});



function refreshStatus() {
    params = new URLSearchParams(window.location.search);
    token = params.get('token')
    
    $.ajax({
        type: "GET",  
        url: "processingstate?token=" + token,
        success: function(data){  
            getStateHandlerFunc(data.athlete_state)(data.athlete_state, data.map_state)
        },
        error: function(XMLHttpRequest, textStatus, errorThrown) { 
            console.log('failure', textStatus, errorThrown);
        }       
    });
}

function getStateHandlerFunc(athlete_state) {
    switch (athlete_state.state) {
        case 'ImportingActivities':
            return handleImportingActivitiesState
        case 'DownloadingActivities':
            return handleDownloadingActivitiesState
        case 'ComputingMapParams':
            return handleComputingMapParamsState
        case 'ProcessingMap':
            return handleProcessingMapState
        default:
            return handleAllOtherStates
    }
}

function handleImportingActivitiesState(athlete_state, map_state) {
    $('#athlete_status').html('Looking for new activities on Strava...')
}

function handleDownloadingActivitiesState(athlete_state, map_state) {
    $('#athlete_status').html('Downloading new activities from Strava...')
}

function handleComputingMapParamsState(athlete_state, map_state) {
    $('#athlete_status').html('Regenerating map metadata. This may take 1-2 minutes...')
}

function handleProcessingMapState(athlete_state, map_state) {
    total = map_state.processing + map_state.failed + map_state.completed
    completePercent = (100 * map_state.completed / total).toFixed(2)
    processingPercent = (100 * map_state.processing / total).toFixed(2)
    failedPercent = (100 * map_state.failed / total).toFixed(2)

    // stop refresh if no more processing
    if (processingPercent == 0) {
        clearInterval(window.refreshTimer)
    }

    if (completePercent == 100) {
        $('#athlete_status').html('Map is fully up to date')
        return
    }

    $('#athlete_status').html('Map is rebuilding. ' + completePercent + '% complete, ' + failedPercent + '% failed. This may be slow at first, but should speed up. Move map around or refresh to see updates.')
}

function handleAllOtherStates(athlete_state, map_state) {
    $('#athlete_status').html('Something may have gone wrong: ' + athlete_state.state)
}