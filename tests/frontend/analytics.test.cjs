const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const source = fs.readFileSync('web/static/js/analytics.js', 'utf8');
function browser({id='G-TEST123',host='pokerplanning.net',storage=new Map()}={}) {
  const listeners = {}, scripts = [];
  const window = {addEventListener:(name,fn)=>listeners[name]=fn};
  const document = {currentScript:{dataset:{measurementId:id}}, title:'Secret team sprint',
    referrer:'https://pokerplanning.net/room/private-referrer-id?name=Alice',
    createElement:()=>({}),head:{append:s=>scripts.push(s)},
    addEventListener:(name, fn)=>listeners[name]=fn};
  vm.runInNewContext(source,{window,document,URL,location:new URL('https://'+host+'/room/private-room-id?invite=private'),
    sessionStorage:{getItem:k=>storage.get(k)||null,setItem:(k,v)=>storage.set(k,v),removeItem:k=>storage.delete(k)}});
  const commands=()=>JSON.parse(JSON.stringify(Array.from(window.dataLayer||[],a=>Array.from(a))));
  return {window,storage,scripts,commands,dispatch:(name,detail={})=>listeners[name]?.({detail}),
    ws:message=>listeners['htmx:wsAfterMessage']?.({detail:{message:JSON.stringify(message)}}),
    events:()=>commands().filter(c=>c[0]==='event')};
}
const state={type:'room_state',payload:{currentParticipantId:'private-participant-id',isCreator:true,roundNumber:1,
  participants:[{ID:'private-participant-id',Name:'Alice private',Role:'voter',Connected:true},
    {ID:'other-private-id',Name:'Bob private',Role:'spectator',Connected:false}]}};

test('canonical bootstrap and every event remove private paths, titles, identities and arbitrary values',()=>{
  const b=browser();
  b.ws(state);
  b.window.pokerAnalytics.track('room_share',{share_method:'copy_link',room_id:'private-room-id',name:'Alice private',value:'secret estimate',page_title:'Secret team sprint'});
  b.window.pokerAnalytics.track('room_share',{share_method:'private URL',participant_role:'Alice private'});
  b.window.pokerAnalytics.track('room_share',JSON.parse('{"constructor":"private","__proto__":"private"}'));
  const output=JSON.stringify(b.commands());
  for(const privateValue of ['private-room-id','private-referrer-id','private-participant-id','Alice','Bob','Secret team','secret estimate','invite=']) assert.ok(!output.includes(privateValue),privateValue);
  const config=b.commands().find(c=>c[0]==='config')[2];
  assert.equal(config.page_location,'https://pokerplanning.net/room/');
  assert.equal(config.page_referrer,'https://pokerplanning.net/');
  assert.equal(config.page_title,'Planning Poker room');
  assert.equal(b.events()[0][2].participant_role,'voter');
  assert.equal(b.events()[0][2].member_type,'host');
  assert.equal(b.events()[0][2].connected_count,1);
  assert.equal(b.scripts.length,1);
});

test('navigation receipts survive the redirect, are sanitized and consumed once',()=>{
  const b=browser();
  b.dispatch('pp:usage',{events:[{name:'room_created',parameters:{deck_type:'custom',name:'private title'}},
    {name:'round_started',parameters:{round_source:'creation',round_number:1}},{name:'unknown',parameters:{}}]});
  assert.equal(b.events().length,0);
  assert.ok(!JSON.stringify([...b.storage]).includes('private title'));
  const next=browser({storage:b.storage});
  assert.deepEqual(next.events().map(e=>e[1]),['room_created','round_started']);
  assert.equal(browser({storage:b.storage}).events().length,0);
});

test('room snapshots and socket reconnects do not replay joins or rounds',()=>{
  const b=browser();
  b.dispatch('htmx:wsOpen'); b.ws(state); b.ws(state);
  b.dispatch('htmx:wsClose'); b.dispatch('htmx:wsClose');
  b.dispatch('htmx:wsOpen'); b.ws(state);
  assert.deepEqual(b.events().map(e=>e[1]),['room_connected','connection_lost','connection_restored']);
  b.ws({type:'votes_revealed',payload:{votes:[{participantName:'Secret',value:'sensitive'}]}});
  assert.equal(b.events().length,3); // Broadcasts never count a mutation for every listener.
  b.ws({type:'usage_event',payload:{name:'round_revealed',parameters:{member_type:'system',reveal_method:'automatic',vote_count:2}}});
  const event=b.events().at(-1);
  assert.equal(event[1],'round_revealed');
  assert.equal(event[2].member_type,'system');
  assert.equal(event[2].participant_role,undefined);
});

test('local and unconfigured builds do not load a Google tag',()=>{
  for(const options of [{host:'localhost'},{id:''},{id:'G-TEST123<script>'}]) {
    const b=browser(options); assert.equal(b.scripts.length,0);assert.equal(b.window.pokerAnalytics,undefined);
  }
});
