// generateInviteCode.go: source of random words for creating invite codes
package site

import (
	"fmt"
	"math/rand"
	"strings"

	"zanaduu3/src/database"
)

// Checks to see if new code is in db before returning it
func GenerateUniqueInviteCode(db *database.DB) (string, error) {
	newCode := GenerateInviteCode()
	var code string
	row := db.NewStatement(`
		SELECT code
		FROM invites
		WHERE code=?
	`).QueryRow(newCode)
	exists, err := row.Scan(&code)
	if exists {
		newCode, err = GenerateUniqueInviteCode(db)
	}
	return newCode, err
}

// Generates a unique invite code of form NOUN1-NOUN2-11847
func GenerateInviteCode() string {
	// Get a random noun
	noun1 := randomNouns[rand.Intn(len(randomNouns))]
	// Get another random noun
	noun2 := randomNouns[rand.Intn(len(randomNouns))]
	// Get a random 5-digit number
	randomNumber := rand.Intn(90000) + 10000
	return strings.ToUpper(fmt.Sprintf("%s-%s-%d", noun1, noun2, randomNumber))
}

var randomNouns = []string{
	"drop",
	"profit",
	"class",
	"prose",
	"expert",
	"market",
	"birthday",
	"flag",
	"silver",
	"sidewalk",
	"van",
	"yak",
	"station",
	"base",
	"shoe",
	"deer",
	"yarn",
	"low",
	"mice",
	"flame",
	"watch",
	"love",
	"jar",
	"cake",
	"tub",
	"butter",
	"glove",
	"surprise",
	"tent",
	"start",
	"planet",
	"galaxy",
	"universe",
	"son",
	"ink",
	"squirrel",
	"earthquake",
	"attraction",
	"trousers",
	"cushion",
	"thunder",
	"spy",
	"hands",
	"thing",
	"baby",
	"cheese",
	"hose",
	"tomatoes",
	"club",
	"cherries",
	"sign",
	"paper",
	"plough",
	"representative",
	"cow",
	"fairies",
	"basketball",
	"zinc",
	"dirt",
	"shirt",
	"tree",
	"cough",
	"head",
	"line",
	"arm",
	"health",
	"relation",
	"quicksand",
	"gold",
	"button",
	"baseball",
	"humor",
	"skate",
	"year",
	"dress",
	"creature",
	"roll",
	"support",
	"knot",
	"harmony",
	"plane",
	"stick",
	"sock",
	"dinosaurs",
	"lunch",
	"thought",
	"amount",
	"care",
	"push",
	"bottle",
	"brass",
	"yam",
	"song",
	"rationality",
	"methods",
	"applied",
	"pleasure",
	"flock",
	"curtain",
	"fish",
	"beds",
	"air",
	"burst",
	"whistle",
	"way",
	"powder",
	"lunchroom",
	"cactus",
	"ladybug",
	"plate",
	"bug",
	"rabbit",
	"governor",
	"bell",
	"statement",
	"quilt",
	"ticket",
	"songs",
	"fog",
	"wool",
	"cabbage",
	"rake",
	"orange",
	"battle",
	"top",
	"grandfather",
	"toes",
	"cave",
	"quiet",
	"spring",
	"stem",
	"bulb",
	"snow",
	"distribution",
	"houses",
	"pan",
	"bear",
	"reason",
	"trains",
	"cent",
	"trick",
	"north",
	"language",
	"fear",
	"door",
	"ducks",
	"structure",
	"bee",
	"substance",
	"toys",
	"division",
	"snail",
	"offer",
	"tooth",
	"arch",
	"authority",
	"middle",
	"seat",
	"slip",
	"mint",
	"bead",
	"sea",
	"bait",
	"shake",
	"sheep",
	"quince",
	"chin",
	"toothpaste",
	"democracy",
	"hair",
	"test",
	"earth",
	"form",
	"grape",
	"partner",
	"appliance",
	"pipe",
	"page",
	"bite",
	"interest",
	"sky",
	"aftermath",
	"ice",
	"bag",
	"sofa",
	"rabbits",
	"mask",
	"scarecrow",
	"front",
	"geese",
	"passenger",
	"grade",
	"stream",
	"writer",
	"driving",
	"brake",
	"measure",
	"chickens",
	"finger",
	"neck",
	"range",
	"boat",
	"weather",
	"vein",
	"quiver",
	"can",
	"birth",
	"pets",
	"lettuce",
	"vegetable",
	"skin",
	"pear",
	"debt",
	"tray",
	"view",
	"horse",
	"ground",
	"seed",
	"stone",
	"cattle",
	"rancher",
	"seashore",
	"stranger",
	"vacation",
	"rings",
	"month",
	"rub",
	"straw",
	"blade",
	"fork",
	"mountain",
	"detail",
	"fang",
	"locket",
	"system",
	"notebook",
	"donkey",
	"fuel",
	"color",
	"approval",
	"transport",
	"pizzas",
	"wheel",
	"dog",
	"action",
	"eye",
	"mist",
	"hospital",
	"produce",
	"dinner",
	"magic",
	"trail",
	"insect",
	"parcel",
	"anger",
	"jam",
	"shelf",
	"sneeze",
	"needle",
	"balance",
	"toothbrush",
	"building",
	"secretary",
	"pail",
	"power",
	"actor",
	"fall",
	"bone",
	"believe",
	"toy",
	"crowd",
	"cart",
	"chalk",
	"holiday",
	"crayon",
	"night",
	"stew",
	"maid",
	"shade",
	"dime",
	"berry",
	"train",
	"army",
	"plants",
	"ants",
	"wax",
	"waves",
	"mother",
	"house",
	"instrument",
	"judge",
	"scissors",
	"belief",
	"treatment",
	"current",
	"stage",
	"school",
	"kick",
	"steam",
	"spiders",
	"bath",
	"railway",
	"guitar",
	"creator",
	"duck",
	"cars",
	"payment",
	"liquid",
	"farm",
	"bay",
	"cove",
	"alcove",
	"arborial",
	"magnolia",
	"primrose",
	"oak",
	"pine",
	"fur",
	"snowflake",
	"acorn",
	"sisters",
	"lake",
	"rose",
	"wood",
	"note",
	"field",
	"square",
	"tax",
	"sand",
	"crate",
	"knife",
	"shock",
	"engine",
	"copper",
	"flowers",
	"guide",
	"argument",
	"spoon",
	"cakes",
	"vessel",
	"friction",
	"sum",
	"kitty",
	"land",
	"discovery",
	"autumn",
	"animal",
	"sleep",
	"writing",
	"sun",
	"airplane",
	"winter",
	"nest",
	"invention",
	"key",
	"sound",
	"rhythm",
	"aunt",
	"oatmeal",
	"bed",
	"collar",
	"circle",
	"grip",
	"pot",
	"need",
	"pickle",
	"teeth",
	"science",
	"floor",
	"brick",
	"room",
	"giants",
	"machine",
	"riddle",
	"rate",
	"drawer",
	"stomach",
	"foot",
	"ocean",
	"stop",
	"scene",
	"umbrella",
	"turkey",
	"letters",
	"hour",
	"alarm",
	"pie",
	"root",
	"frog",
	"slave",
	"work",
	"shape",
	"use",
	"fly",
	"water",
	"walk",
	"trip",
	"trees",
	"wound",
	"slope",
	"afternoon",
	"lip",
	"plastic",
	"coach",
	"muscle",
	"title",
	"dogs",
	"match",
	"curve",
	"connection",
	"lamp",
	"taste",
	"bridge",
	"horn",
	"hook",
	"desire",
	"zipper",
	"yoke",
	"yolk",
	"home",
	"stove",
	"ring",
	"knowledge",
	"sleet",
	"mark",
	"toad",
	"weight",
	"end",
	"party",
	"wall",
	"swing",
	"soap",
	"uncle",
	"hammer",
	"turn",
	"porter",
	"kiss",
	"business",
	"frame",
	"smash",
	"girls",
	"morning",
	"thumb",
	"digestion",
	"oranges",
	"bikes",
	"condition",
	"order",
	"achiever",
	"skirt",
	"meeting",
	"pet",
	"flight",
	"underwear",
	"word",
	"veil",
	"distance",
	"milk",
	"increase",
	"wire",
	"cap",
	"fold",
	"scarf",
	"change",
	"jewel",
	"silk",
	"impulse",
	"camp",
	"bells",
	"territory",
	"church",
	"egg",
	"agreement",
	"property",
	"sister",
	"decision",
	"bike",
	"twig",
	"marble",
	"cream",
	"stamp",
	"quartz",
	"bedroom",
	"boundary",
	"advice",
	"competition",
	"stocking",
	"stretch",
	"meal",
	"degree",
	"mass",
	"smile",
	"week",
	"jail",
	"back",
	"picture",
	"hand",
	"apparatus",
	"tongue",
	"selection",
	"elbow",
	"harbor",
	"minister",
	"throne",
	"pen",
	"theory",
	"honey",
	"canvas",
	"boot",
	"sink",
	"friends",
	"daughter",
	"receipt",
	"town",
	"story",
	"sponge",
	"snails",
	"thread",
	"dolls",
	"run",
	"society",
	"calculator",
	"afterthought",
	"touch",
	"basin",
	"limit",
	"gate",
	"crack",
	"cows",
	"behavior",
	"side",
	"lock",
	"pies",
	"country",
	"cellar",
	"tank",
	"chance",
	"account",
	"giraffe",
	"dock",
	"plant",
	"fireman",
	"corn",
	"fire",
	"grandmother",
	"clam",
	"expansion",
	"hat",
	"rain",
	"education",
	"memory",
	"eggnog",
	"idea",
	"roof",
	"grass",
	"coast",
	"friend",
	"whip",
	"rock",
	"amusement",
	"poison",
	"monkey",
	"women",
	"yard",
	"discussion",
	"existence",
	"road",
	"zoo",
	"time",
	"clover",
	"respect",
	"talk",
	"jellyfish",
	"route",
	"self",
	"ray",
	"company",
	"event",
	"crook",
	"books",
	"book",
	"sticks",
	"experience",
	"stitch",
	"hill",
	"shoes",
	"shop",
	"servant",
	"cracker",
	"money",
	"potato",
	"twist",
	"acoustics",
	"metal",
	"bubble",
	"finger",
	"man",
	"pigs",
	"haircut",
	"rainstorm",
	"motion",
	"kettle",
	"quarter",
	"chess",
	"look",
	"cat",
	"attack",
	"oil",
	"cats",
	"spot",
	"carpenter",
	"peace",
	"spark",
	"oven",
	"silence",
	"insurance",
	"trouble",
	"star",
	"horses",
	"cover",
	"hall",
	"design",
	"laborer",
	"rice",
	"wind",
	"snake",
	"committee",
	"doctor",
	"cub",
	"income",
	"drain",
	"sail",
	"eggs",
	"sack",
	"chicken",
	"letter",
	"quill",
	"trade",
	"rat",
	"nation",
	"island",
	"addition",
	"direction",
	"playground",
	"library",
	"office",
	"texture",
	"wheeze",
	"wizard",
	"hobbit",
	"dwarf",
	"elf",
	"nothing",
	"nothingness",
	"suchness",
	"id",
	"ego",
	"superego",
	"animus",
	"creativity",
	"panda",
	"creation",
	"maker",
	"baker",
	"thrill",
	"border",
	"volleyball",
	"visitor",
	"scale",
	"car",
	"store",
	"string",
	"suit",
	"birds",
	"crown",
	"move",
	"babies",
	"nose",
	"soup",
	"hobbies",
	"comparison",
	"mouth",
	"plot",
	"verse",
	"cook",
	"badge",
	"pancake",
	"beginner",
	"box",
	"reaction",
	"question",
	"cloth",
	"woman",
	"wilderness",
	"feeling",
	"glass",
	"ear",
	"industry",
	"temper",
	"coat",
	"waste",
	"trucks",
	"brother",
	"wine",
	"adjustment",
	"calendar",
	"moon",
	"smoke",
	"value",
	"recess",
	"bucket",
	"cannon",
	"vest",
	"legs",
	"voyage",
	"iron",
	"card",
	"river",
	"sweater",
	"wealth",
	"rifle",
	"smell",
	"blow",
	"things",
	"steel",
	"rail",
	"window",
	"table",
	"act",
	"art",
	"effect",
	"place",
	"heat",
	"coal",
	"step",
	"throat",
	"arithmetic",
	"toe",
	"fruit",
	"observation",
	"shame",
	"desk",
	"bat",
	"religion",
	"vase",
	"cable",
	"planes",
	"angle",
	"mind",
	"nerve",
	"frogs",
	"noise",
	"scent",
	"swim",
	"camel",
	"cocoa",
	"coconut",
	"other",
	"externality",
	"furniture",
	"rule",
	"bit",
	"camera",
	"mailbox",
	"pull",
	"purpose",
	"edge",
	"history",
	"pin",
	"person",
	"cobweb",
	"example",
	"pencil",
	"record",
	"pollution",
	"basket",
	"beef",
	"wren",
	"food",
	"zephyr",
	"join",
	"soda",
	"cherry",
	"fact",
	"robin",
	"downtown",
	"face",
	"name",
	"hope",
	"branch",
	"development",
	"tiger",
	"crow",
	"government",
	"leg",
	"jeans",
	"carriage",
	"team",
	"icicle",
	"hole",
	"celery",
	"bushes",
	"lace",
	"apparel",
	"reading",
	"grain",
	"sugar",
	"level",
	"spade",
	"force",
	"hot",
	"screw",
	"crib",
	"teaching",
	"lumber",
	"cause",
	"airport",
	"cup",
	"control",
	"truck",
	"price",
	"voice",
	"mine",
	"worm",
	"dust",
	"space",
	"leather",
	"tin",
	"doll",
	"credit",
	"cast",
	"channel",
	"children",
	"pig",
	"loaf",
	"coil",
	"knee",
	"sheet",
	"rest",
	"bird",
	"protest",
	"wash",
	"request",
	"street",
	"linen",
	"ghost",
	"pocket",
	"size",
	"laugh",
	"mitten",
	"ship",
	"board",
	"mom",
	"day",
	"men",
	"flower",
	"jump",
	"kittens",
	"wing",
	"wave",
	"exchange",
	"sense",
	"hydrant",
	"jelly",
	"wish",
	"tail",
	"part",
	"wrench",
	"volcano",
	"position",
	"unit",
	"fowl",
	"wrist",
	"play",
	"breath",
	"eyes",
	"number",
	"activity",
	"point",
	"show",
	"snakes",
	"suggestion",
	"zebra",
	"group",
	"tendency",
	"popcorn",
	"growth",
	"flavor",
	"sort",
	"juice",
	"queen",
	"dad",
	"summer",
	"salt",
	"reward",
	"drink",
	"caption",
	"minute",
}
