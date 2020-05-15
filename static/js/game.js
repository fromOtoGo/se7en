socket = new WebSocket("ws://127.0.0.1:8000/wsgame");
var isFirstCard = false
var isFirstTurnOfRound = false
// отправка bet из формы
document.forms.publish.onsubmit = function() {
  let outgoingMessage = this.message.value;
  let message1 = {
    "bet": outgoingMessage
  }
  socket.send(JSON.stringify(message1));
  return false;
};
//send chosen card
function imgsrc(num){
  let message = {
    "card_number": num
  }
  socket.send(JSON.stringify(message));
}
//send joker
function joker(num){
  let message = {
    "card_number": num
  }
  socket.send(JSON.stringify(message));
  let su =  ["♥", "♦", "♣", "♠"]
  //sadddddddddddddddddddddddd
  for (i=0;i<4;i++){
    div = document.createElement('div')
    div.id = "joker"
    div.innerHTML = div.innerHTML = '<p class="jok"> <img src="/static/Cards/'+su[i]+'.jpg"'+' class="jok'+(20+i*2)+'" onClick="JokerByMax('+(+i+4)+')";/> </p>';;
    document.body.append(div);
  }
//sadddddddddddddddddddddddd
  for (i=0;i<1;i++){
    div = document.createElement('div')
    div.id = "joker"
    div.innerHTML = '<p class="jok"> <img src="/static/Cards/'+"min"+i+'.jpg"'+' class="jok'+(18+i*2)+'" onClick="sendJoker('+(+i+4)+')";/> </p>';
    document.body.append(div);
    div = document.createElement('div')
    div.id = "joker"
    div.innerHTML = '<p class="jok"> <img src="/static/Cards/'+"max"+i+'.jpg"'+' class="jok'+(19+i*2)+'" onClick="sendJoker('+i+')";/> </p>';
    document.body.append(div);
  }
}
//send charasterisitc of joker
function sendJoker(jok){
  var mean
  // isFirstCard = true
  // isFirstTurnOfRound = true
  if (jok == 4){
    
    mean = 4
    if(isFirstCard){
      if (isFirstTurnOfRound){
        for (i=0;i<2;i++){
          var div = document.getElementById("joker");
          if (div != undefined) {
            div.remove();
           }
        }
        let su =  ["♥", "♦", "♣", "♠"]
        for (i=0;i<4;i++){
          div = document.createElement('div')
          div.id = "joker"
          div.innerHTML = div.innerHTML = '<p class="jok"> <img src="/static/Cards/'+su[i]+'.jpg"'+' class="jok'+(18+i*2)+'" onClick="JokerByMax('+(+i+4)+')";/> </p>';;
          document.body.append(div);
        }
      }
      return
    }
  }else if(jok == 0){
    mean = 0
    if(isFirstCard){
      if (isFirstTurnOfRound){
        for (i=0;i<2;i++){
          var div = document.getElementById("joker");
          if (div != undefined) {
            div.remove();
           }
        }
        let su =  ["♥", "♦", "♣", "♠"]
        for (i=0;i<4;i++){
          div = document.createElement('div')
          div.id = "joker"
          div.innerHTML = '<p class="jok"> <img src="/static/Cards/'+ su[i] +"9"+'.jpg"'+' class="jok'+(19+i*2)+'" onClick="JokerByMax('+(+i+1)+')";/> </p>';
          document.body.append(div);
        }
        return
      }
   }
  }
  alert(mean)
  let message = {
    "joker": mean
  }
  socket.send(JSON.stringify(message));
  for (i=0;i<10;i++){
    var div = document.getElementById("joker");
    if (div != undefined) {
	    div.remove();
 	  }
  }
  
}
function JokerByMax(code){
  let message = {
    "joker": code
  }
  socket.send(JSON.stringify(message));
  for (i=0;i<10;i++){
    var div = document.getElementById("joker");
    if (div != undefined) {
	    div.remove();
 	  }
  }
}

// получение сообщения
socket.onmessage = function(event) {
  let message = event.data;
  let messageElem = document.createElement('div');
  parsed = JSON.parse(message)
  //show names in table
  if (parsed.names){
    // alert(parsed.turn)
    var table = "";
    table+='<table>'
    table+='<tr><th> </th>'
    for (i in parsed.names){
      table+='<th colspan="2">'+parsed.names[i]+'</th>'
    }
    table+="</tr>"
    if (parsed.totalScore){
      table+='<tr><th> </th>'
      for (i in parsed.totalScore){
        table+='<th colspan="2">'+parsed.totalScore[i]+'</th>'
      }
      table+= '<tr><td>№</td>'
      for(i in parsed.totalScore){
        table+= '<td>bet</td><td>got</td>'
      }
      table+= '</tr>'
    }
    
    //show score table
    if (parsed.scoreChart){
      for(i in parsed.scoreChart){
        let rnd = (+1)+(+i)
        table+='<tr><td>'+rnd+'</td>'
        for (j in parsed.scoreChart[i]){
          table+='<td>'+parsed.scoreChart[i][j].Bet+'</td>'
          table+='<td>'+parsed.scoreChart[i][j].Got+'</td>'
        }
        table+='</tr>'
        //BLOCK SHOWS CARDS TAKEN, NAMES AND AVATARS
        let lastindex = 0
        for (i in parsed.scoreChart){
          lastindex = i
        }
        var indexWithShift
        for (num in parsed.names){
          playersCount = +num+1
        }
        for (j in parsed.scoreChart[lastindex]){
          indexWithShift = +j + parsed.position
          if(indexWithShift == playersCount){
            indexWithShift -= playersCount
          }
          var div = document.getElementById("plName"+indexWithShift);
          if (div != undefined) {
            div.remove();
          }
          div = document.createElement('div')
          div.id = "plName"+indexWithShift
          div.innerHTML = '<div class="name"><div class="name'+j+'"><H2>'+parsed.names[indexWithShift]+'</H2></div></div>'
          document.body.append(div);

          var div = document.getElementById("avatar"+indexWithShift);
          if (div != undefined) {
            div.remove();
          }
          var div = document.getElementById("taken"+indexWithShift);
          if (div != undefined) {
            div.remove();
          }
          var div = document.getElementById("takenCnt"+indexWithShift);
          if (div != undefined) {
            div.remove();
          }
          div = document.createElement('div')
          div.id = "avatar"+indexWithShift
          div.innerHTML = '<div class="avatar"><img src="/static/avatars/gopher'+j+'.jpg" + class="avatar'+indexWithShift+'";/></div>'
          document.body.append(div);
          
          if(+parsed.scoreChart[lastindex][j].Got == 1){
            div = document.createElement('div')
            div.id = "taken"+indexWithShift
            div.innerHTML = '<div class="takenCards"> <img src="/static/Cards/suit.jpg" + class="takenCards'+indexWithShift+'";/></div>'
            document.body.append(div);
          }else if (+parsed.scoreChart[lastindex][j].Got > 1){
            div = document.createElement('div')
            div.id = "taken"+indexWithShift
            div.innerHTML = '<div class="takenCards"> <img src="/static/Cards/suit.jpg" + class="takenCards'+indexWithShift+'";/></div>'
            document.body.append(div);

            div = document.createElement('div')
            div.id = "takenCnt"+indexWithShift
            div.innerHTML = '<div class="takenCount"><div class="takenCount'+indexWithShift+'"><h1>'+parsed.scoreChart[lastindex][j].Got+'</h1></div> '
            document.body.append(div);
          }
        }
        //show border on next turn
        let turn = parsed.position + parsed.turn
        let len = 0
        for (i in parsed.names){
          len++
          var div = document.getElementById("border"+i);
          if (div != undefined) {
            div.remove();
          }
        }
        if (turn >= len){
          turn-=len
        }
        
        div = document.createElement('div')
        div.id = "border"+turn
        div.innerHTML = '<div class="rpos'+turn+'"><div class="ramka-1"></div></div>'
        document.body.append(div);
        //END BLOCK
        
      //show border when expecting bet
      }
      var div = document.getElementById("betposition");
      if (div != undefined) {
        div.remove();
      }
      if (parsed.isBet == true){
        div = document.createElement('div')
        div.id = "betposition"
        div.innerHTML = '<div class="betpos"><div class="ramka-5-wr"></div></div>'
        document.body.append(div);
      }
    }
    table+='</table>'
 
  
    var div = document.getElementById("table");
    if (div != undefined) {
      div.remove();
    }

    div = document.createElement('div')
    div.id = "table"
    div.innerHTML = table
    document.body.append(div);
  }
  if (parsed.trump){
    var div = document.getElementById("trump");
    if (div != undefined) {
	    div.remove();
    }
    var div = document.createElement('div')
    div.id = "trump"
    div.innerHTML = '<p class="regularCard"> <img src="/static/Cards/'+parsed.trump+'.jpg'+'" class="trump" /> </p>';
    document.body.append(div);
  }
  
  len = 0;
  for (i in parsed.cards){
    len++;
  }
  if (len > 0){
      for (i=0;i<18;i++){
        var div = document.getElementById("card");
        if (div != undefined) {
          div.remove();
        }
      }
      for (i=0;i<6;i++){
        var div = document.getElementById("player");
        if (div != undefined) {
          div.remove();
        }
      }
    for (i in parsed.cards){
      if (parsed.cards[i] == "0000"){
        div = document.createElement('div')
        div.id = "card"
        div.innerHTML = '<p class="card"> <img src="/static/Cards/'+'min0'+'.jpg"'+' class="cards'+i+'" onClick="imgsrc('+i+')";/> </p>';
        document.body.append(div);
      }else if (parsed.cards[i] == "♠1"){
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
  }
  

  len = 0;
  for (i in parsed.player){
    len++;
  }
  if (parsed.player){
    
    for (i in parsed.player){
      
      div = document.createElement('div')
      div.id = "player"
      div.innerHTML = '<p class="regularCard"> <img src="/static/Cards/'+parsed.player[i]+'.jpg'+'" class="xplayer'+i+'" /> </p>';
      document.body.append(div);
    }
  }


}

