-- Database: PortfolioApis

-- DROP DATABASE "PortfolioApis";

/*CREATE DATABASE "PortfolioApis"
    WITH 
    OWNER = postgres
    ENCODING = 'UTF8'
    LC_COLLATE = 'English_United States.1252'
    LC_CTYPE = 'English_United States.1252'
    TABLESPACE = pg_default
    CONNECTION LIMIT = -1;

COMMENT ON DATABASE "PortfolioApis"
    IS 'Database which holds scripts and users data';*/
	
-- Table: public.companies

-- DROP TABLE public.companies;

CREATE TABLE IF NOT EXISTS public.companies
(
    company_id character varying(30) COLLATE pg_catalog."default" NOT NULL,
    company_name character varying(100) COLLATE pg_catalog."default" NOT NULL,
    load_date date,
    CONSTRAINT companies_pkey PRIMARY KEY (company_id)
)

TABLESPACE pg_default;

ALTER TABLE public.companies
    OWNER to postgres;
	
-- Table: public.companies_price_data

-- DROP TABLE public.companies_price_data;

CREATE TABLE IF NOT EXISTS public.companies_price_data
(
    company_id character varying(30) COLLATE pg_catalog."default" NOT NULL,
    open_val numeric(30,10),
    high_val numeric(30,10),
    low_val numeric(30,10),
    close_val numeric(30,10),
    date_val date NOT NULL,
    CONSTRAINT companies_price_data_pkey PRIMARY KEY (company_id, date_val)
)

TABLESPACE pg_default;

ALTER TABLE public.companies_price_data
    OWNER to postgres;
	
-- Table: public.user_holdings

-- DROP TABLE public.user_holdings;

CREATE TABLE IF NOT EXISTS public.user_holdings
(
    user_id character varying(30) COLLATE pg_catalog."default",
    company_id character varying(30) COLLATE pg_catalog."default",
    quantity numeric(30,10),
    buy_date date,
    buy_price numeric(30,10)
)

TABLESPACE pg_default;

ALTER TABLE public.user_holdings
    OWNER to postgres;
	
-- Table: public.user_holdings_nt

-- DROP TABLE public.user_holdings_nt;

CREATE TABLE IF NOT EXISTS public.user_holdings_nt
(
    user_id character varying(30) COLLATE pg_catalog."default",
    security_id character varying(30) COLLATE pg_catalog."default",
    buy_date date,
    buy_value numeric(30,10),
    current_value numeric(30,10),
    interest_rate numeric(10,2)
)

TABLESPACE pg_default;

ALTER TABLE public.user_holdings_nt
    OWNER to postgres;
	
-- Table: public.user_model_pf

-- DROP TABLE public.user_model_pf;

CREATE TABLE IF NOT EXISTS public.user_model_pf
(
    user_id character varying(30) COLLATE pg_catalog."default" NOT NULL,
    security_id character varying(30) COLLATE pg_catalog."default" NOT NULL,
    reasonable_price numeric(30,10),
    exp_alloc numeric(5,2),
    CONSTRAINT user_model_pf_pkey PRIMARY KEY (user_id, security_id)
)

TABLESPACE pg_default;

ALTER TABLE public.user_model_pf
    OWNER to postgres;
	
-- Table: public.users

-- DROP TABLE public.users;

CREATE TABLE IF NOT EXISTS public.users
(
    user_id character varying(30) COLLATE pg_catalog."default" NOT NULL,
    start_date date,
    exp_eq_alloc numeric(5,2),
    target_amount numeric(30,10),
    CONSTRAINT users_pkey PRIMARY KEY (user_id)
)

alter table users add column password character varying(500)

TABLESPACE pg_default;

ALTER TABLE public.users
    OWNER to postgres;