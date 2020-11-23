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
    $('#status_icon').attr('src', '/static/icons/queue_black_48dp.png')
    $('#status_text').html('Importing activities...')
}

function handleDownloadingActivitiesState(athlete_state, map_state) {
    $('#status_icon').attr('src', '/static/icons/cloud_download_black_48dp.png')
    $('#status_text').html('Downloading activities...')
}

function handleComputingMapParamsState(athlete_state, map_state) {
    $('#status_icon').attr('src', '/static/icons/speed_black_48dp.png')
    $('#status_text').html('Computing map parameters...')
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
        $('#status_icon').attr('src', '/static/icons/verified_black_48dp.png')
        $('#status_text').html('Up to date!')
        return
    }

    $('#status_icon').attr('src', '/static/icons/speed_black_48dp.png')
    $('#status_text').html('Rebuilding - ' + completePercent + '% complete. May be slow at first but will speed up. Move around or refresh to see updates.')
}

function handleAllOtherStates(athlete_state, map_state) {
    $('#athlete_status').html('Something may have gone wrong: ' + athlete_state.state)
}