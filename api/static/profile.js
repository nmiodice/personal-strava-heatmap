function init() {
    console.log('profile loaded')
    $('#syncbutton').prop('disabled', true)
    $('#summary').html("")


    window.params = new URLSearchParams(window.location.search);
    token = window.params.get('token')

    console.log('token:', token)
    if (token == null || token == "") {
        alert('Did not find auth token! Please login...')
        window.location = location.protocol + '//' + location.host
        return
    }

    $('#message').html("Loading Profile...")

    $.ajax({
        url: "unprocessedactivities?" + window.params.toString(),
        success: function(result){
            console.log(result)
            $('#message').html("Profile Loaded...")
            $('#summary').append('<tr><td>Total Strava Activites</td><td>Newly Found Activities</td><td>Unsynced Activities</tr>')
            $('#summary').append(
                '<tr>' + 
                '<td>' + result.ActivityRefresh.Total + '</td>' +
                '<td>' + result.ActivityRefresh.New.length + '</td>' +
                '<td>' + result.ActivityRefresh.Unsynced.length + '</td>' +
                '</tr>')

            if (result.ActivityRefresh.Unsynced.length > 0) {
                $('#syncbutton').click(triggerSync)
                $('#syncbutton').prop('disabled', false)
            }
        }
    });
}


function triggerSync() {
    $('#syncbutton').prop('disabled', true)
    $('#message').html("Syncing Activities...")

    $.ajax({
        url: "syncactivities?" + window.params.toString(),
        success: function(result){
            console.log(result)
            init()
        }
    });
}

$( document ).ready(init);
