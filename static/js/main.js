socket = new WebSocket("ws://127.0.0.1:8000/wsmain");




document.forms.publish.onsubmit = function() {
    let gameName = document.forms.game_pass.login.value;
    let gamePass = document.forms.game_pass.password.value;

    let e = document.getElementById("players");
    let players = e.options[e.selectedIndex].value;

    let message1 = {
        "name": gameName,
        "pass": gamePass,
        "max_players": players,
    }
  
    socket.send(JSON.stringify(message1));
    return false;
};


var objSel = document.getElementById("myGames");

socket.onmessage = function(event) {
  objSel.options.length = 0;
  let message = event.data;
  

  //   if (parsed.game_id != undefined){
  //   alert(document.location.host)
  //  }

  parsed = JSON.parse(message);
  for(i in parsed){
    let name = parsed[i].name + " " + parsed[i].players+"/"+parsed[i].maxpl
    objSel.options[objSel.options.length] = new Option(name, parsed[i].id);
    
  }
  if(parsed.game_id){
    document.location.href = "/game?table="+parsed.game_id;
  }
  
}

document.forms.join.onsubmit = function() {
  let game = document.getElementById("myGames");
  let id = game.options[game.selectedIndex].value;
  let message1 = {
    "join": id
  }

  socket.send(JSON.stringify(message1));
  return false;
};
