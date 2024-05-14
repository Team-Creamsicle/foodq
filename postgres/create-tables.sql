create type mealtype as ENUM ('BREAKFAST','LUNCH','DINNER','DESSERT','SNACK');
create type ingredienttype as ENUM ('VEGETABLE','FRUIT','MEAT','MEATALT','FATS','DAIRY','GRAIN','SPICE','PULSE','OTHER');

create table Users (
	id SERIAL PRIMARY KEY,
	name varchar(100),
	email varchar(100) NOT NULL,
	createdate timestamp DEFAULT (now() at time zone 'utc'),
	lastlogin timestamp DEFAULT (now() at time zone 'utc'),
	subscription boolean
);

create table Queues (
	id SERIAL PRIMARY KEY,
	userid integer references Users(id),
	queue integer ARRAY,
	createdate timestamp DEFAULT (now() at time zone 'utc'),
	updatedate timestamp DEFAULT (now() at time zone 'utc')
);

create table Ingredients (
	id SERIAL PRIMARY KEY,
	name varchar(64),
	category ingredienttype NOT NULL,
	createdate timestamp DEFAULT (now() at time zone 'utc'),
	updatedate timestamp DEFAULT (now() at time zone 'utc')
);

create table Recipes (
	id SERIAL PRIMARY KEY,
	name varchar(64) NOT NULL,
	instructions varchar(1000) NOT NULL,
	ingredients varchar(1000) NOT NULL,
	category mealtype,
	cuisinetype varchar(40),
    servings smallint,
	dietaryrestrictions varchar(1000),
	totaltime interval,
	preptime interval,
	cooktime interval,
	creatorId integer references Users(id),
	createdate timestamp DEFAULT (now() at time zone 'utc'),
	updatedate timestamp DEFAULT (now() at time zone 'utc')
);

create table SavedRecipes (
	id SERIAL PRIMARY KEY,
	userid integer references Users(id),
	recipeid integer references Recipes(id)
);

create index index_ingredient_category on Ingredients(category);
create index index_recipe_category on Recipes(category);
create index index_recipe_cuisine_type on Recipes(cuisinetype);
create index index_recipe_cuisine_type on Recipes(cuisinetype);
create index index_queue_userid on Queues(userid)