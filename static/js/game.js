socket = new WebSocket("ws://127.0.0.1:8000/wsgame");


// отправка сообщения из формы
document.forms.publish.onsubmit = function() {
  let outgoingMessage = this.message.value;

  let message1 = {
    "bet": outgoingMessage
  }

  socket.send(JSON.stringify(message1));
  return false;
};

function imgsrc(num){
  let message = {
    "card_number": num
  }
  socket.send(JSON.stringify(message));
}

function joker(num){
  let message = {
    "card_number": num
  }
  socket.send(JSON.stringify(message));
  for (i=0;i<5;i++){
    div = document.createElement('div')
    div.id = "joker"
    div.innerHTML = '<p class="card"> <img src="/static/Cards/'+"min"+i+'.jpg"'+' class="cards'+(18+i*2)+'" onClick="sendJoker('+i+')";/> </p>';
    document.body.append(div);
    div = document.createElement('div')
    div.id = "joker"
    div.innerHTML = '<p class="card"> <img src="/static/Cards/'+"max"+i+'.jpg"'+' class="cards'+(19+i*2)+'" onClick="sendJoker('+i+')";/> </p>';
    document.body.append(div);
  }

}

function sendJoker(jok){
  let message = {
    "joker": jok
  }
  alert(jok)
  socket.send(JSON.stringify(message));
  for (i=0;i<10;i++){
    var div = document.getElementById("joker");
    if (div != undefined) {
	    div.remove();
 	  }
  }
}

// получение сообщения - отобразить данные в div#messages
socket.onmessage = function(event) {
  let message = event.data;
  let messageElem = document.createElement('div');
  parsed = JSON.parse(message)
  messageElem.textContent = parsed.name;
  // document.getElementById('messages').prepend(messageElem);
  
  // let message = {
  //   "cardNumber": num
  // }
  
  // socket.send(JSON.stringify(message));

  var div = document.getElementById("trump");
  if (div != undefined) {
	  div.remove();
  }
  var div = document.createElement('div')
  div.id = "trump"
  if (parsed.trump){
    div.innerHTML = '<p class="card"> <img src="/static/Cards/'+parsed.trump+'.jpg'+'" class="trump" /> </p>';
    document.body.append(div);
  }

  len = 0;
  for (i in parsed.cards){
    len++;
  }

  for (i=0;i<18;i++){
    var div = document.getElementById("card");
    if (div != undefined) {
	    div.remove();
 	  }
  }

  for (i in parsed.cards){
    if (parsed.cards[i] == "♠1"){
      div = document.createElement('div')
      div.id = "card"
      div.innerHTML = '<p class="card"> <img src="/static/Cards/'+parsed.cards[i]+'.jpg"'+' class="cards'+i+'" onClick="joker('+i+')";/> </p>';
      document.body.append(div);
    }else{
      div = document.createElement('div')
      div.id = "card"
      div.innerHTML = '<p class="card"> <img src="/static/Cards/'+parsed.cards[i]+'.jpg"'+' class="cards'+i+'" onClick="imgsrc('+i+')";/> </p>';
      document.body.append(div);
    }
  }

  len = 0;
  for (i in parsed.cards){
    len++;
  }

  for (i=0;i<6;i++){
    var div = document.getElementById("player");
    if (div != undefined) {
	    div.remove();
 	  }
  }
  // document.getElementById('messages').prepend(parsed.player[i]);
  for (i in parsed.player){
    
    div = document.createElement('div')
    div.id = "player"
    div.innerHTML = '<p class="card"> <img src="/static/Cards/'+parsed.player[i]+'.jpg'+'" class="player'+i+'" /> </p>';
    document.body.append(div);
  }
}

