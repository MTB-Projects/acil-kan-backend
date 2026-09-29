#!/usr/bin/env node
// OpenStreetMap'ten il/ilçe ve hastane listesini üretir.
//
//   node scripts/osm/refresh.js [--app ../acil-kan-main]
//
// Çıktılar:
//   internal/hospital/hospitals.json     (backend'e gömülür, /public/hospitals)
//   <app>/assets/data/districts.json     (uygulamaya gömülür, ilçe otomatik tamamlama)
//
// Overpass API'ye nazik davranır: istekler sırayla gider, hız sınırında bekler,
// ham yanıtlar scripts/osm/.cache altında saklanır (yeniden çalıştırınca tekrar indirilmez;
// taze veri için klasörü silin). Veri © OpenStreetMap katkıcıları, ODbL.
'use strict';
const fs = require('fs');
const path = require('path');
const { execFileSync } = require('child_process');

const ROOT = path.resolve(__dirname, '..', '..');
const CACHE = path.join(__dirname, '.cache');
const APP = path.resolve(ROOT, argValue('--app') || '../acil-kan-main');
const ENDPOINT = 'https://overpass-api.de/api/interpreter';

// OSM adı -> uygulamadaki il adı
const PROVINCE_FIX = { 'Elâzığ': 'Elazığ', 'Hakkâri': 'Hakkari' };
// OSM'de yanlışlıkla ilçe (admin_level 6) olarak işaretlenmiş yerler
const NOT_DISTRICTS = new Set(['Kara Ada']);
// İnsan hastanesi olmayanlar
const EXCLUDE_NAME = /veteriner|hayvan/i;

function argValue(flag) {
  const i = process.argv.indexOf(flag);
  return i > 0 ? process.argv[i + 1] : null;
}

function sleep(s) { execFileSync('sleep', [String(s)]); }

function overpass(name, query) {
  fs.mkdirSync(CACHE, { recursive: true });
  const file = path.join(CACHE, name + '.json');
  if (fs.existsSync(file)) return JSON.parse(fs.readFileSync(file, 'utf8'));
  for (let attempt = 1; attempt <= 6; attempt++) {
    let body = '';
    try {
      body = execFileSync('curl', ['-s', '-m', '600', '-A', 'acil-kan-osm-refresh/1.0',
        '--data-urlencode', 'data=' + query, ENDPOINT], { maxBuffer: 256 << 20 }).toString();
    } catch (e) { body = 'curl: ' + e.message; }
    if (body.trim().startsWith('{')) {
      fs.writeFileSync(file, body);
      return JSON.parse(body);
    }
    const reason = body.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').slice(-100);
    console.error(`  ${name}: deneme ${attempt} başarısız (${reason.trim()})`);
    sleep(20 * attempt);
  }
  throw new Error(`${name}: Overpass yanıt vermedi`);
}

const TR = 'area["ISO3166-1"="TR"][admin_level=2]->.tr;';

// --- 1) İller ve ilçeler ---
function loadDistricts() {
  console.log('İller ve ilçeler indiriliyor…');
  // a) İlin alanına dokunan tüm ilçeler (sınır komşuları da gelir)
  const all = overpass('districts_all', `[out:json][timeout:600];${TR}
    rel(area.tr)["admin_level"="4"]["boundary"="administrative"]->.provs;
    foreach.provs->.p(.p out tags;.p map_to_area->.a;
      rel(area.a)["admin_level"="6"]["boundary"="administrative"];out tags;);`);
  // b) Merkez noktası ilin içinde olan ilçeler (komşuları eler, merkezi tanımsız olanları kaçırır)
  const centred = overpass('districts_centred', `[out:json][timeout:600];${TR}
    rel(area.tr)["admin_level"="4"]["boundary"="administrative"]->.provs;
    foreach.provs->.p(.p out tags;.p map_to_area->.a;
      rel(area.a)["admin_level"="6"]["boundary"="administrative"]->.ds;
      (node(r.ds:"admin_centre")(area.a);node(r.ds:"label")(area.a);)->.c;
      rel(bn.c)["admin_level"="6"]["boundary"="administrative"];out tags;);`);

  const group = (res) => {
    const m = new Map(); let cur = null;
    for (const e of res.elements) {
      const t = e.tags || {};
      if (t.admin_level === '4') { cur = PROVINCE_FIX[t.name] || t.name; if (!m.has(cur)) m.set(cur, new Map()); }
      else if (cur && t.admin_level === '6') m.get(cur).set(e.id, t.name);
    }
    return m;
  };
  const A = group(all), C = group(centred);
  const centredIds = new Set([...C.values()].flatMap((d) => [...d.keys()]));
  const seenIn = new Map();
  for (const [p, ds] of A) for (const [id] of ds) seenIn.set(id, [...(seenIn.get(id) || []), p]);

  const out = {};
  const add = (p, name) => { if (!NOT_DISTRICTS.has(name)) (out[p] = out[p] || new Set()).add(name); };
  for (const [p, ds] of C) for (const [, name] of ds) add(p, name);
  // Merkezi tanımsız ilçeler: yalnızca tek bir ilin alanında görünüyorsa o ile aittir
  for (const [p, ds] of A) for (const [id, name] of ds) {
    if (!centredIds.has(id) && seenIn.get(id).length === 1) add(p, name);
  }

  const provinces = Object.keys(out).filter((p) => /^[A-Za-zÇĞİÖŞÜçğıöşüÂâÎîÛû ]+$/.test(p));
  const result = {};
  for (const p of provinces.sort((a, b) => a.localeCompare(b, 'tr'))) {
    result[p] = [...out[p]].sort((a, b) => a.localeCompare(b, 'tr'));
  }
  const count = Object.values(result).reduce((n, d) => n + d.length, 0);
  console.log(`  ${Object.keys(result).length} il, ${count} ilçe`);
  if (Object.keys(result).length !== 81) throw new Error('81 il bekleniyordu');
  return result;
}

// --- 2) Hastaneler ---
function loadHospitals(districts) {
  console.log('Hastaneler indiriliyor…');
  const res = overpass('hospitals', `[out:json][timeout:300];${TR}
    nwr["amenity"="hospital"]["name"](area.tr);out center tags;`);
  const points = res.elements
    .map((e) => ({ e, key: e.type[0] + e.id, lat: e.lat ?? e.center?.lat, lon: e.lon ?? e.center?.lon }))
    .filter((p) => p.lat != null);
  console.log(`  ${points.length} hastane; il/ilçeleri bulunuyor…`);

  // Her nokta için "hangi il ve ilçede?" (is_in), 120'lik gruplar halinde
  const where = new Map();
  const B = 120;
  for (let b = 0; b * B < points.length; b++) {
    const parts = points.slice(b * B, (b + 1) * B).map((p) =>
      `make h key="${p.key}";out;is_in(${p.lat},${p.lon})->.a;` +
      `area.a["admin_level"~"^(4|6)$"]["boundary"="administrative"];out tags;`);
    const r = overpass(`is_in_${b}`, '[out:json][timeout:300];' + parts.join(''));
    let key = null;
    for (const e of r.elements) {
      if (e.type === 'h') { key = e.tags.key; where.set(key, {}); continue; }
      const t = e.tags || {};
      if (t.admin_level === '4') where.get(key).province = PROVINCE_FIX[t.name] || t.name;
      if (t.admin_level === '6') (where.get(key).districts ||= []).push(t.name);
    }
    sleep(3);
  }

  const seen = new Set();
  const list = [];
  for (const p of points) {
    const t = p.e.tags;
    const name = (t['name:tr'] || t.name || '').trim();
    const w = where.get(p.key) || {};
    const city = w.province;
    if (!name || !city || !districts[city] || EXCLUDE_NAME.test(name)) continue;
    const district = (w.districts || []).find((d) => districts[city].includes(d)) || '';
    const dedupe = `${city}|${district}|${name.toLocaleLowerCase('tr')}`;
    if (seen.has(dedupe)) continue;
    seen.add(dedupe);
    const street = [t['addr:street'], t['addr:housenumber']].filter(Boolean).join(' No:');
    const address = t['addr:full'] || [t['addr:neighbourhood'] || t['addr:suburb'], street].filter(Boolean).join(', ');
    list.push({
      id: p.key, name, city, district,
      ...(address ? { address } : {}),
      lat: Math.round(p.lat * 1e6) / 1e6,
      lng: Math.round(p.lon * 1e6) / 1e6,
    });
  }
  list.sort((a, b) => a.city.localeCompare(b.city, 'tr') || a.name.localeCompare(b.name, 'tr'));
  const withDistrict = list.filter((h) => h.district).length;
  console.log(`  ${list.length} hastane yazılacak (${withDistrict} tanesinin ilçesi belli)`);
  return list;
}

const districts = loadDistricts();
const hospitals = loadHospitals(districts);

const hospitalFile = path.join(ROOT, 'internal', 'hospital', 'hospitals.json');
fs.writeFileSync(hospitalFile, JSON.stringify(hospitals, null, 0).replace(/\},\{/g, '},\n{') + '\n');
console.log('Yazıldı:', path.relative(process.cwd(), hospitalFile));

const districtFile = path.join(APP, 'assets', 'data', 'districts.json');
if (fs.existsSync(APP)) {
  fs.mkdirSync(path.dirname(districtFile), { recursive: true });
  fs.writeFileSync(districtFile, JSON.stringify(districts, null, 1) + '\n');
  console.log('Yazıldı:', path.relative(process.cwd(), districtFile));
} else {
  console.log('Uygulama klasörü bulunamadı, districts.json yazılmadı:', APP);
}
